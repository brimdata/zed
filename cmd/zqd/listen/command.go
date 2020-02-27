package listen

import (
	"flag"
	"net"
	"net/http"
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
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.listenAddr, "l", ":9867", "[addr]:port to listen on")
	f.StringVar(&c.dataDir, "datadir", ".", "data directory")
	return c, nil
}

func (c *Command) Run(args []string) error {
	dataDir, err := filepath.Abs(c.dataDir)
	if err != nil {
		return err
	}
	logger := newLogger()
	handler := zqd.NewHandler(dataDir)
	ln, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return err
	}
	logger.Info("Listening",
		zap.String("addr", ln.Addr().String()),
		zap.String("datadir", dataDir),
	)
	return http.Serve(ln, handler)
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
