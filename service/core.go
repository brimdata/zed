package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zson"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// DefaultZedFormat is the default Zed format that the server will assume if the
// value for a request's "Accept" or "Content-Type" headers are not set or set
// to "*/*".
const DefaultZedFormat = "zson"

const indexPage = `
<!DOCTYPE html>
<html>
  <title>Zed lake service</title>
  <body style="padding:10px">
    <h2>zed serve</h2>
    <p>A <a href="https://github.com/brimdata/zed/tree/main/cmd/zed/serve">zed service</a> is listening on this host/port.</p>
    <p>If you're a <a href="https://www.brimdata.io/">Brim</a> user, connect to this host/port from the <a href="https://github.com/brimdata/brim">Brim application</a> in the graphical desktop interface in your operating system (not a web browser).</p>
    <p>If your goal is to perform command line operations against this Zed lake, use the <a href="https://github.com/brimdata/zed/tree/main/cmd/zed"><code>zed</code></a> command.</p>
  </body>
</html>`

type Config struct {
	Auth        AuthConfig
	Logger      *zap.Logger
	Root        *storage.URI
	RootContent io.ReadSeeker
	Version     string
}

type Core struct {
	auth            *Auth0Authenticator
	conf            Config
	engine          storage.Engine
	logger          *zap.Logger
	registry        *prometheus.Registry
	root            *lake.Root
	routerAPI       *mux.Router
	routerAux       *mux.Router
	taskCount       int64
	subscriptions   map[chan event]struct{}
	subscriptionsMu sync.RWMutex
}

func NewCore(ctx context.Context, conf Config) (*Core, error) {
	if conf.Logger == nil {
		conf.Logger = zap.NewNop()
	}
	if conf.RootContent == nil {
		conf.RootContent = strings.NewReader(indexPage)
	}
	if conf.Version == "" {
		conf.Version = "unknown"
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())

	var authenticator *Auth0Authenticator
	if conf.Auth.Enabled {
		var err error
		if authenticator, err = NewAuthenticator(ctx, conf.Logger, registry, conf.Auth); err != nil {
			return nil, err
		}
	}
	path := conf.Root
	if path == nil {
		return nil, errors.New("no lake root")
	}
	var engine storage.Engine
	switch storage.Scheme(path.Scheme) {
	case storage.FileScheme:
		engine = storage.NewLocalEngine()
	case storage.S3Scheme:
		engine = storage.NewRemoteEngine()
	default:
		return nil, fmt.Errorf("root path cannot have scheme %q", path.Scheme)
	}
	root, err := lake.CreateOrOpen(ctx, engine, path)
	if err != nil {
		return nil, err
	}

	routerAux := mux.NewRouter()
	routerAux.Use(corsMiddleware())

	routerAux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "", time.Time{}, conf.RootContent)
	})

	debug := routerAux.PathPrefix("/debug/pprof").Subrouter()
	debug.HandleFunc("/cmdline", pprof.Cmdline)
	debug.HandleFunc("/profile", pprof.Profile)
	debug.HandleFunc("/symbol", pprof.Symbol)
	debug.HandleFunc("/trace", pprof.Trace)
	debug.PathPrefix("/").HandlerFunc(pprof.Index)

	routerAux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	routerAux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	})
	routerAux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&api.VersionResponse{Version: conf.Version})
	})

	routerAPI := mux.NewRouter()
	routerAPI.Use(requestIDMiddleware())
	routerAPI.Use(accessLogMiddleware(conf.Logger))
	routerAPI.Use(panicCatchMiddleware(conf.Logger))
	routerAPI.Use(corsMiddleware())

	c := &Core{
		auth:          authenticator,
		conf:          conf,
		engine:        engine,
		logger:        conf.Logger.Named("core"),
		root:          root,
		registry:      registry,
		routerAPI:     routerAPI,
		routerAux:     routerAux,
		subscriptions: make(map[chan event]struct{}),
	}

	c.addAPIServerRoutes()
	c.logger.Info("Started")
	return c, nil
}

func (c *Core) addAPIServerRoutes() {
	c.authhandle("/auth/identity", handleAuthIdentityGet).Methods("GET")
	// /auth/method intentionally requires no authentication
	c.routerAPI.Handle("/auth/method", c.handler(handleAuthMethodGet)).Methods("GET")
	c.authhandle("/events", handleEvents).Methods("GET")
	c.authhandle("/index", handleIndexRulesDelete).Methods("DELETE")
	c.authhandle("/index", handleIndexRulesPost).Methods("POST")
	c.authhandle("/pool", handlePoolPost).Methods("POST")
	c.authhandle("/pool/{pool}", handlePoolDelete).Methods("DELETE")
	c.authhandle("/pool/{pool}", handleBranchPost).Methods("POST")
	c.authhandle("/pool/{pool}", handlePoolPut).Methods("PUT")
	c.authhandle("/pool/{pool}/branch/{branch}", handleBranchGet).Methods("GET")
	c.authhandle("/pool/{pool}/branch/{branch}", handleBranchDelete).Methods("DELETE")
	c.authhandle("/pool/{pool}/branch/{branch}", handleBranchLoad).Methods("POST")
	c.authhandle("/pool/{pool}/branch/{branch}/delete", handleDelete).Methods("POST")
	c.authhandle("/pool/{pool}/branch/{branch}/index", branchHandle(handleIndexApply)).Methods("POST")
	c.authhandle("/pool/{pool}/branch/{branch}/index/update", branchHandle(handleIndexUpdate)).Methods("POST")
	c.authhandle("/pool/{pool}/branch/{branch}/merge/{child}", handleBranchMerge).Methods("POST")
	c.authhandle("/pool/{pool}/branch/{branch}/revert/{commit}", handleRevertPost).Methods("POST")
	c.authhandle("/pool/{pool}/stats", handlePoolStats).Methods("GET")
	c.authhandle("/query", handleQuery).Methods("OPTIONS", "POST")
}

func (c *Core) handler(f func(*Core, *ResponseWriter, *Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, req := newRequest(w, r, c.logger)
		f(c, res, req)
	})
}

func (c *Core) authhandle(path string, f func(*Core, *ResponseWriter, *Request)) *mux.Route {
	if c.auth != nil {
		f = c.auth.Middleware(f)
	}
	return c.routerAPI.Handle(path, c.handler(f))
}

func branchHandle(f func(*Core, *ResponseWriter, *Request, *lake.Branch)) func(*Core, *ResponseWriter, *Request) {
	return func(c *Core, w *ResponseWriter, r *Request) {
		poolID, ok := r.PoolID(w, c.root)
		if !ok {
			return
		}
		branchName, ok := r.StringFromPath(w, "branch")
		if !ok {
			return
		}
		pool, err := c.root.OpenPool(r.Context(), poolID)
		if err != nil {
			w.Error(err)
			return
		}
		branch, err := pool.OpenBranchByName(r.Context(), branchName)
		if err != nil {
			w.Error(err)
			return
		}
		f(c, w, r, branch)
	}
}

func (c *Core) Registry() *prometheus.Registry {
	return c.registry
}

func (c *Core) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rm mux.RouteMatch
	if c.routerAux.Match(r, &rm) {
		rm.Handler.ServeHTTP(w, r)
		return
	}
	c.routerAPI.ServeHTTP(w, r)
}

func (c *Core) Shutdown() {
	c.logger.Info("Shutdown")
}

func (c *Core) nextTaskID() int64 {
	return atomic.AddInt64(&c.taskCount, 1)
}

func (c *Core) requestLogger(r *http.Request) *zap.Logger {
	return c.logger.With(zap.String("request_id", api.RequestIDFromContext(r.Context())))
}

func (c *Core) publishEvent(w *ResponseWriter, name string, data interface{}) {
	zv, err := zson.MarshalZNG(data)
	if err != nil {
		w.Logger.Error("Error marshaling published event", zap.Error(err))
		return
	}
	go func() {
		ev := event{name: name, value: &zv}
		c.subscriptionsMu.RLock()
		for sub := range c.subscriptions {
			sub <- ev
		}
		c.subscriptionsMu.RUnlock()
	}()
}
