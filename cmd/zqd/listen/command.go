package listen

import (
	"flag"
	"net"
	"net/http"
	"net/http/pprof"
	"path/filepath"

	"github.com/brimsec/zq/cmd/zqd/root"
	"github.com/brimsec/zq/zqd"
	"github.com/mccanne/charm"
	"go.uber.org/zap"
)

var Listen = &charm.Spec{
	Name:  "listen",
	Usage: "listen [options]",
	Short: "listen as a daemon and repond to zqd service requests",
	Long: `
The listen command launches a process to listen on the provided interface and
`,
	New: New,
}

func init() {
	root.Zqd.Add(Listen)
}

type Command struct {
	*root.Command
	listenAddr string
	dataDir    string
	zeekExec   string
	pprof      bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	f.StringVar(&c.dataDir, "datadir", ".", "data directory")
	f.StringVar(&c.zeekExec, "zeekpath", "", "path to the zeek executable to use (defaults to zeek in $PATH)")
	f.BoolVar(&c.pprof, "pprof", false, "add pprof routes to api")
	return c, nil
}

func (c *Command) pprofHandlers(h http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func (c *Command) Run(args []string) error {
	dataDir, err := filepath.Abs(c.dataDir)
	if err != nil {
		return err
	}
	logger := newLogger()
	core := &zqd.Core{Root: dataDir, ZeekExec: c.zeekExec}
	h := zqd.NewHandler(core)
	if c.pprof {
		h = c.pprofHandlers(h)
	}
	ln, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return err
	}
	logger.Info("Listening",
		zap.String("addr", ln.Addr().String()),
		zap.String("datadir", dataDir),
		zap.Bool("pprof_routes", c.pprof),
	)
	return http.Serve(ln, h)
}

func newLogger() *zap.Logger {
	c := zap.NewProductionConfig()
	c.Sampling = nil
	c.EncoderConfig.CallerKey = ""
	l, err := c.Build()
	if err != nil {
		panic(err)
	}
	return l
}
