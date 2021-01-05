package zqd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"sync/atomic"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/apiserver"
	"github.com/brimsec/zq/ppl/zqd/pcapanalyzer"
	"github.com/brimsec/zq/ppl/zqd/recruiter"
	"github.com/brimsec/zq/ppl/zqd/worker"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const indexPage = `
<!DOCTYPE html>
<html>
  <title>ZQD daemon</title>
  <body style="padding:10px">
    <h2>ZQD</h2>
    <p>A <a href="https://github.com/brimsec/zq/tree/master/cmd/zqd">zqd</a> daemon is listening on this host/port.</p>
    <p>If you're a <a href="https://www.brimsecurity.com/">Brim</a> user, connect to this host/port from the <a href="https://github.com/brimsec/brim">Brim application</a> in the graphical desktop interface in your operating system (not a web browser).</p>
    <p>If your goal is to perform command line operations against this zqd, use the <a href="https://github.com/brimsec/zq/tree/master/cmd/zapi">zapi</a> client.</p>
  </body>
</html>`

type Config struct {
	Auth        AuthConfig
	DB          apiserver.DBConfig
	Logger      *zap.Logger
	Personality string
	Root        string
	Version     string
	Worker      worker.WorkerConfig

	Suricata pcapanalyzer.Launcher
	Zeek     pcapanalyzer.Launcher
}

type middleware interface {
	Middleware(next http.Handler) http.Handler
}

type Core struct {
	auth       *Auth0Authenticator
	logger     *zap.Logger
	mgr        *apiserver.Manager
	registry   *prometheus.Registry
	root       iosrc.URI
	router     *mux.Router
	taskCount  int64
	workerPool *recruiter.WorkerPool     // state for personality=recruiter
	workerReg  *worker.RegistrationState // state for personality=worker
	worker     worker.WorkerConfig       // config for personality=worker

	suricata pcapanalyzer.Launcher
	zeek     pcapanalyzer.Launcher
}

func NewCore(ctx context.Context, conf Config) (*Core, error) {
	if conf.Logger == nil {
		conf.Logger = zap.NewNop()
	}
	if conf.Version == "" {
		conf.Version = "unknown"
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())

	root, err := iosrc.ParseURI(conf.Root)
	if err != nil {
		return nil, err
	}

	mgr, err := apiserver.NewManager(ctx, conf.Logger, registry, root, conf.DB)
	if err != nil {
		return nil, err
	}

	var auth *Auth0Authenticator
	if conf.Auth.Enabled {
		if auth, err = newAuthenticator(ctx, conf.Logger, registry, conf.Auth); err != nil {
			return nil, err
		}
	}

	router := mux.NewRouter()
	router.Use(requestIDMiddleware())
	router.Use(accessLogMiddleware(conf.Logger))
	router.Use(panicCatchMiddleware(conf.Logger))
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, indexPage)
	})
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	})
	router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&api.VersionResponse{Version: conf.Version})
	})

	c := &Core{
		auth:     auth,
		logger:   conf.Logger,
		mgr:      mgr,
		registry: registry,
		root:     root,
		router:   router,
		suricata: conf.Suricata,
		worker:   conf.Worker,
		zeek:     conf.Zeek,
	}

	switch conf.Personality {
	case "", "all":
		c.addAPIServerRoutes()
		c.addWorkerRoutes()
	case "apiserver":
		c.addAPIServerRoutes()
	case "recruiter":
		c.workerPool = recruiter.NewWorkerPool()
		c.addRecruiterRoutes()
	case "worker":
		c.addWorkerRoutes()
	default:
		return nil, fmt.Errorf("unknown personality %s", conf.Personality)
	}

	return c, nil
}

func (c *Core) addAPIServerRoutes() {
	c.authhandle("/ast", handleASTPost).Methods("POST")

	// /auth/method intentionally requires no authentication
	c.router.Handle("/auth/method", c.handler(handleAuthMethodGet)).Methods("GET")
	c.authhandle("/auth/identity", handleAuthIdentityGet).Methods("GET")

	c.authhandle("/search", handleSearch).Methods("POST")
	c.authhandle("/space", handleSpaceList).Methods("GET")
	c.authhandle("/space", handleSpacePost).Methods("POST")
	c.authhandle("/space/{space}", handleSpaceDelete).Methods("DELETE")
	c.authhandle("/space/{space}", handleSpaceGet).Methods("GET")
	c.authhandle("/space/{space}", handleSpacePut).Methods("PUT")
	c.authhandle("/space/{space}/archivestat", handleArchiveStat).Methods("GET")
	c.authhandle("/space/{space}/index", handleIndexPost).Methods("POST")
	c.authhandle("/space/{space}/indexsearch", handleIndexSearch).Methods("POST")
	c.authhandle("/space/{space}/log", handleLogStream).Methods("POST")
	c.authhandle("/space/{space}/log/paths", handleLogPost).Methods("POST")
	c.authhandle("/space/{space}/pcap", handlePcapPost).Methods("POST")
	c.authhandle("/space/{space}/pcap", handlePcapSearch).Methods("GET")
	c.authhandle("/space/{space}/subspace", handleSubspacePost).Methods("POST")
}

func (c *Core) addRecruiterRoutes() {
	c.router.Handle("/recruiter/listfree", c.handler(handleListFree)).Methods("GET")
	c.router.Handle("/recruiter/recruit", c.handler(handleRecruit)).Methods("POST")
	c.router.Handle("/recruiter/register", c.handler(handleRegister)).Methods("POST")
	c.router.Handle("/recruiter/stats", c.handler(handleRecruiterStats)).Methods("GET")
}

func (c *Core) addWorkerRoutes() {
	c.router.Handle("/worker/chunksearch", c.handler(handleWorkerChunkSearch)).Methods("POST")
	c.router.Handle("/worker/release", c.handler(handleWorkerRelease)).Methods("GET")
	c.router.Handle("/worker/rootsearch", c.handler(handleWorkerRootSearch)).Methods("POST")
}

func (c *Core) handler(f func(*Core, http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f(c, w, r)
	})
}

func (c *Core) authhandle(path string, f func(*Core, http.ResponseWriter, *http.Request)) *mux.Route {
	var h http.Handler
	if c.auth != nil {
		h = c.auth.Middleware(c.handler(f))
	} else {
		h = c.handler(f)
	}
	return c.router.Handle(path, h)
}

func (c *Core) HTTPHandler() http.Handler {
	return c.router
}

func (c *Core) HasSuricata() bool {
	return c.suricata != nil
}

func (c *Core) HasZeek() bool {
	return c.zeek != nil
}

func (c *Core) Registry() *prometheus.Registry {
	return c.registry
}

func (c *Core) Root() iosrc.URI {
	return c.root
}

func (c *Core) Shutdown() {
	c.mgr.Shutdown()
}

func (c *Core) nextTaskID() int64 {
	return atomic.AddInt64(&c.taskCount, 1)
}

func (c *Core) requestLogger(r *http.Request) *zap.Logger {
	return c.logger.With(zap.String("request_id", getRequestID(r.Context())))
}

func (c *Core) WorkerRegistration(ctx context.Context, srvAddr string, conf worker.WorkerConfig) error {
	if _, _, err := net.SplitHostPort(conf.Recruiter); err != nil {
		return errors.New("flag -worker.recruiter=host:port must be provided for -personality=worker")
	}
	if conf.Node == "" {
		return errors.New("flag -worker.node must be provided for -personality=worker")
	}
	var err error
	c.workerReg, err = worker.NewRegistrationState(ctx, srvAddr, conf, c.logger)
	if err != nil {
		return err
	}
	go c.workerReg.RegisterWithRecruiter()
	return nil
}
