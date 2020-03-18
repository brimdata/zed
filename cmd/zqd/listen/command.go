package listen

import (
	"errors"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"path/filepath"

	"github.com/brimsec/zq/cmd/zqd/logger"
	"github.com/brimsec/zq/cmd/zqd/root"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/zeek"
	"github.com/mccanne/charm"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

// defaultLogger ignores output from the access logger.
var defaultLogger = []logger.Config{
	{
		Name:  "zqd",
		Path:  "stderr",
		Level: zap.InfoLevel,
	},
}

func init() {
	root.Zqd.Add(Listen)
}

type Command struct {
	*root.Command
	listenAddr string
	conf       zqd.Config
	pprof      bool
	zeekpath   string
	configfile string
	loggerConf []logger.Config
	devMode    bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	f.StringVar(&c.conf.Root, "datadir", ".", "data directory")
	f.StringVar(&c.zeekpath, "zeekpath", "", "path to the zeek executable to use (defaults to zeek in $PATH)")
	f.BoolVar(&c.pprof, "pprof", false, "add pprof routes to api")
	f.StringVar(&c.configfile, "config", "", "path to a zqd config file")
	f.BoolVar(&c.devMode, "dev", false, "runs zqd in development mode")
	return c, nil
}

func (c *Command) Run(args []string) error {
	var err error
	c.conf.Root, err = filepath.Abs(c.conf.Root)
	if err != nil {
		return err
	}
	if err := c.loadConfigFile(); err != nil {
		return err
	}
	logger, err := c.logger()
	if err != nil {
		return err
	}
	c.conf.Logger = logger.Named("zqd")
	if err := c.loadzeek(); err != nil {
		return err
	}
	core := zqd.NewCore(c.conf)
	h := zqd.NewHandlerWithLogger(core, logger)
	if c.pprof {
		h = pprofHandlers(h)
	}
	ln, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return err
	}
	c.conf.Logger.Info("Listening",
		zap.String("addr", ln.Addr().String()),
		zap.String("datadir", c.conf.Root),
		zap.Bool("pprof_routes", c.pprof),
		zap.Bool("zeek_supported", core.HasZeek()),
	)
	return http.Serve(ln, h)
}

func pprofHandlers(h http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// Example configfile
// loggers:
// - path: ./data/access.log
//   name: "http.access"
//   level: info
//   mode: truncate

func (c *Command) loadConfigFile() error {
	if c.configfile == "" {
		return nil
	}
	// For now config file just has loggers.
	conf := struct {
		Loggers []logger.Config `yaml:"loggers"`
	}{}
	b, err := ioutil.ReadFile(c.configfile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return err
	}
	c.loggerConf = conf.Loggers
	return err
}

func (c *Command) loadzeek() error {
	ln, err := zeek.LauncherFromPath(c.zeekpath)
	if err != nil && !errors.Is(err, zeek.ErrNotFound) {
		return err
	}
	c.conf.ZeekLauncher = ln
	return nil
}

func (c *Command) logger() (*zap.Logger, error) {
	if c.loggerConf == nil {
		c.loggerConf = defaultLogger
	}
	core, err := logger.New(c.loggerConf...)
	if err != nil {
		return nil, err
	}
	// If the development mode is on, calls to logger.DPanic will cause a panic
	// whereas in production would result in an error.
	var opts []zap.Option
	if c.devMode {
		opts = append(opts, zap.Development())
	}
	return zap.New(core, opts...), nil
}
