package serve

import (
	"context"
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/httpd"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/service"
	"github.com/brimdata/zed/service/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Cmd = &charm.Spec{
	Name:  "serve",
	Usage: "serve [options]",
	Short: "service requests to a Zed lake",
	Long: `
The serve command listens for Zed lake API requests on the provided
interface and port, executes the requests, and returns results.
Requests may be issued to this service via the "zed api" command.
`,
	HiddenFlags: "brimfd,filestorereadonly,nodename,podip,recruiter,workers",
	New:         New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	conf    service.Config
	logger  *zap.Logger
	logconf logger.Config

	// brimfd is a file descriptor passed through by brim desktop. If set the
	// command will exit if the fd is closed.
	brimfd          int
	devMode         bool
	listenAddr      string
	portFile        string
	rootContentFile string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.conf.Auth.SetFlags(f)
	c.conf.Version = cli.Version
	f.IntVar(&c.brimfd, "brimfd", -1, "pipe read fd passed by brim to signal brim closure")
	f.BoolVar(&c.devMode, "dev", false, "run in development mode")
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	f.Var(&c.logconf.Level, "log.level", "logging level")
	f.StringVar(&c.logconf.Path, "log.path", "stderr", "logging level")
	c.logconf.Mode = logger.FileModeTruncate
	f.Var(&c.logconf.Mode, "log.filemode", "logger file write mode (values: append, truncate, rotate)")
	f.StringVar(&c.portFile, "portfile", "", "write listen port to file")
	f.StringVar(&c.rootContentFile, "rootcontentfile", "", "file to serve for GET /")
	c.LakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	uri, err := c.URI()
	if err != nil {

	}
	if api.IsLakeService(uri) {
		return errors.New("serve command available for local lakes only")
	}
	c.conf.Root = uri
	if c.rootContentFile != "" {
		f, err := fs.Open(c.rootContentFile)
		if err != nil {
			return err
		}
		defer f.Close()
		c.conf.RootContent = f
	}
	if err := c.initLogger(); err != nil {
		return err
	}
	defer c.logger.Sync()
	openFilesLimit, err := rlimit.RaiseOpenFilesLimit()
	if err != nil {
		c.logger.Warn("Raising open files limit failed", zap.Error(err))
	}
	c.conf.Logger.Info("Open files limit raised", zap.Uint64("limit", openFilesLimit))

	if c.brimfd != -1 {
		if ctx, err = c.watchBrimFd(ctx); err != nil {
			return err
		}
	}
	core, err := service.NewCore(ctx, c.conf)
	if err != nil {
		return err
	}
	defer core.Shutdown()
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		sig := <-sigch
		c.logger.Info("Signal received", zap.Stringer("signal", sig))
	}()
	srv := httpd.New(c.listenAddr, core)
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

func (c *Command) watchBrimFd(ctx context.Context) (context.Context, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("flag -brimfd not applicable to windows")
	}
	f := os.NewFile(uintptr(c.brimfd), "brimfd")
	c.logger.Info("Listening to brim process pipe", zap.String("fd", f.Name()))
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		io.Copy(io.Discard, f)
		c.logger.Info("Brim fd closed, shutting down")
		cancel()
	}()
	return ctx, nil
}

func (c *Command) initLogger() error {
	core, err := logger.NewCore(c.logconf)
	if err != nil {
		return err
	}
	opts := []zap.Option{zap.AddStacktrace(zapcore.WarnLevel)}
	// If the development mode is on, calls to logger.DPanic will cause a panic
	// whereas in production would result in an error.
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
