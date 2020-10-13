package listen

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"runtime"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/cmd/zqd/logger"
	"github.com/brimsec/zq/cmd/zqd/root"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/httpd"
	"github.com/brimsec/zq/pkg/rlimit"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/pcapanalyzer"
	"github.com/mccanne/charm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var Listen = &charm.Spec{
	Name:  "listen",
	Usage: "listen [options]",
	Short: "listen as a daemon and repond to zqd service requests",
	Long: `
The listen command launches a process to listen on the provided interface and
`,
	HiddenFlags: "brimfd",
	New:         New,
}

func init() {
	root.Zqd.Add(Listen)
}

type Command struct {
	*root.Command
	listenAddr         string
	conf               zqd.Config
	pprof              bool
	prom               bool
	suricataRunnerPath string
	zeekRunnerPath     string
	configfile         string
	loggerConf         *logger.Config
	logLevel           zapcore.Level
	logger             *zap.Logger
	devMode            bool
	portFile           string
	// brimfd is a file descriptor passed through by brim desktop. If set zqd
	// will exit if the fd is closed.
	brimfd int
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.conf.Version = cli.Version
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	f.StringVar(&c.conf.Root, "data", ".", "data location")
	f.StringVar(&c.suricataRunnerPath, "suricatarunner", "", "path to command that generates suricata eve.json from pcap data")
	f.StringVar(&c.zeekRunnerPath, "zeekrunner", "", "path to command that generates zeek logs from pcap data")
	f.BoolVar(&c.pprof, "pprof", false, "add pprof routes to api")
	f.BoolVar(&c.prom, "prometheus", false, "add prometheus metrics routes to api")
	f.StringVar(&c.configfile, "config", "", "path to a zqd config file")
	f.Var(&c.logLevel, "loglevel", "level for log output (defaults to info)")
	f.BoolVar(&c.devMode, "dev", false, "runs zqd in development mode")
	f.StringVar(&c.portFile, "portfile", "", "write port of http listener to file")

	// hidden
	f.IntVar(&c.brimfd, "brimfd", -1, "pipe read fd passed by brim to signal brim closure")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if err := c.init(); err != nil {
		return err
	}
	c.logger.Info("init complete")
	openFilesLimit, err := rlimit.RaiseOpenFilesLimit()
	if err != nil {
		c.logger.Warn("Raising open files limit failed", zap.Error(err))
	}
	c.logger.Info("rlimit.raised")
	core, err := zqd.NewCore(c.conf)
	if err != nil {
		return err
	}
	c.logger.Info("Starting",
		zap.String("datadir", c.conf.Root),
		zap.Uint64("open_files_limit", openFilesLimit),
		zap.Bool("pprof_routes", c.pprof),
		zap.Bool("suricata_supported", core.HasSuricata()),
		zap.Bool("zeek_supported", core.HasZeek()),
	)
	h := zqd.NewHandler(core, c.logger)
	if c.pprof {
		h = pprofHandlers(h)
	}
	if c.prom {
		h = prometheusHandlers(h)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if c.brimfd != -1 {
		if ctx, err = c.watchBrimFd(ctx); err != nil {
			return err
		}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		sig := <-ch
		c.logger.Info("Signal received", zap.Stringer("signal", sig))
		cancel()
	}()
	srv := httpd.New(c.listenAddr, h)
	srv.SetLogger(c.logger.Named("httpd"))
	if err := srv.Start(ctx); err != nil {
		return err
	}
	if c.portFile != "" {
		if err := c.writePortFile(srv.Addr()); err != nil {
			return err
		}
	}
	return srv.Wait()
}

func (c *Command) init() error {
	if err := c.loadConfigFile(); err != nil {
		return err
	}
	if err := c.initLogger(); err != nil {
		return err
	}
	if err := c.initZeek(); err != nil {
		return err
	}
	return c.initSuricata()
}

func (c *Command) watchBrimFd(ctx context.Context) (context.Context, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("flag -brimfd not applicable to windows")
	}
	f := os.NewFile(uintptr(c.brimfd), "brimfd")
	c.logger.Info("Listening to brim process pipe", zap.String("fd", f.Name()))
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		io.Copy(ioutil.Discard, f)
		c.logger.Info("Brim fd closed, shutting down")
		cancel()
	}()
	return ctx, nil
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

// XXX Eventually this function should take prometheus.Registry as an argument.
// For now since we only care about retrieving go stats, create registry
// here.
func prometheusHandlers(h http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", h)
	promreg := prometheus.NewRegistry()
	promreg.MustRegister(prometheus.NewGoCollector())
	promhandler := promhttp.HandlerFor(promreg, promhttp.HandlerOpts{})
	mux.Handle("/metrics", promhandler)
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
// sort_mem_max_bytes: 268432640

func (c *Command) loadConfigFile() error {
	if c.configfile == "" {
		return nil
	}
	conf := &struct {
		Logger          logger.Config `yaml:"logger"`
		SortMemMaxBytes *int          `yaml:"sort_mem_max_bytes,omitempty"`
	}{}
	b, err := ioutil.ReadFile(c.configfile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, conf)
	c.loggerConf = &conf.Logger
	if v := conf.SortMemMaxBytes; v != nil {
		if *v <= 0 {
			return fmt.Errorf("%s: sortMemMaxBytes value must be greater than zero", c.configfile)
		}
		sort.MemMaxBytes = *v
	}

	return err
}

func (c *Command) initZeek() error {
	if c.zeekRunnerPath == "" {
		var err error
		if c.zeekRunnerPath, err = exec.LookPath("zeekrunner"); err != nil {
			return nil
		}
	}
	ln, err := pcapanalyzer.LauncherFromPath(c.zeekRunnerPath)
	if err != nil {
		return err
	}
	c.conf.Zeek = ln
	return nil
}

func (c *Command) initSuricata() error {
	if c.suricataRunnerPath == "" {
		var err error
		if c.suricataRunnerPath, err = exec.LookPath("suricatarunner"); err != nil {
			return nil
		}
	}
	ln, err := pcapanalyzer.LauncherFromPath(c.suricataRunnerPath)
	if err != nil {
		return err
	}
	c.conf.Suricata = ln
	return nil
}

// defaultLogger ignores output from the access logger.
func (c *Command) defaultLogger() *logger.Config {
	return &logger.Config{
		Type: logger.TypeWaterfall,
		Children: []logger.Config{
			{
				Name:  "http.access",
				Path:  "/dev/null",
				Level: c.logLevel,
			},
			{
				Path:  "stderr",
				Level: c.logLevel,
			},
		},
	}
}

func (c *Command) initLogger() error {
	if c.loggerConf == nil {
		c.loggerConf = c.defaultLogger()
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

func (c *Command) writePortFile(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	return fs.ReplaceFile(c.portFile, 0644, func(w io.Writer) error {
		_, err := w.Write([]byte(port))
		return err
	})
}
