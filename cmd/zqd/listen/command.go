package listen

import (
	"errors"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/cmd/zqd/logger"
	"github.com/brimsec/zq/cmd/zqd/root"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/zeek"
	"github.com/looky-cloud/boom/pkg/catcher"
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
var defaultLogger = &logger.Config{
	Type: logger.TypeWaterfall,
	Children: []logger.Config{
		{
			Name:  "http.access",
			Path:  "/dev/null",
			Level: zap.InfoLevel,
		},
		{
			Path:  "stderr",
			Level: zap.InfoLevel,
		},
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
	loggerConf *logger.Config
	logger     *zap.Logger
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
	if err := c.init(); err != nil {
		return err
	}
	core := zqd.NewCore(c.conf)
	c.logger.Info("Starting",
		zap.String("datadir", c.conf.Root),
		zap.Bool("pprof_routes", c.pprof),
		zap.Bool("zeek_supported", core.HasZeek()),
	)
	h := zqd.NewHandler(core, c.logger)
	if c.pprof {
		h = pprofHandlers(h)
	}
	ln, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return err
	}
	c.logger.Info("Listening", zap.Stringer("addr", ln.Addr()))
	return http.Serve(ln, h)
}

func (c *Command) init() error {
	var err error
	c.conf.Root, err = filepath.Abs(c.conf.Root)
	if err != nil {
		return err
	}
	if err := c.initLogger(); err != nil {
		return err
	}
	c.catcher = catcher.NewSignalCatcher(os.Interrupt)
	c.catcher.SetOnCaught(func(sig os.Signal) {
		c.logger.Info("Received signal", zap.String("signal", sig.String()))
	})
	return c.initZeek()
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
// logger:
//   type: waterfall
//   children:
//   - path: ./data/access.log
//     name: "http.access"
//     level: info
//     mode: truncate

func (c *Command) loadConfigFile() error {
	if c.configfile == "" {
		return nil
	}
	// For now config file just has logger.
	conf := struct {
		Logger *logger.Config `yaml:"logger"`
	}{Logger: c.loggerConf}
	b, err := ioutil.ReadFile(c.configfile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, &conf)
}

func (c *Command) initZeek() error {
	ln, err := zeek.LauncherFromPath(c.zeekpath)
	if err != nil && !errors.Is(err, zeek.ErrNotFound) {
		return err
	}
	c.conf.ZeekLauncher = ln
	return nil
}

func (c *Command) initLogger() error {
	if c.loggerConf == nil {
		c.loggerConf = defaultLogger
	}
	core, err := logger.NewCore(*c.loggerConf)
	if err != nil {
		return err
	}
	// If the development mode is on, calls to logger.DPanic will cause a panic
	// whereas in production would result in an error.
	var opts []zap.Option
	if c.devMode {
		opts = append(opts, zap.Development())
	}
	c.logger = zap.New(core, opts...)
	c.conf.Logger = c.logger
	return nil
}
