package zqd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func NewRedisClient(ctx context.Context, logger *zap.Logger, conf RedisConfig) (*redis.Client, error) {
	if !conf.Enabled {
		return nil, nil
	}

	logger = logger.Named("redis")
	client := redis.NewClient(&conf.Options)
	client.AddHook(redisotel.TracingHook{})

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Could not connect to server",
			zap.String("url", conf.StringRedacted()),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Info("Connected", zap.String("url", conf.StringRedacted()))
	return client, nil
}

type RedisConfig struct {
	redis.Options

	Enabled      bool
	PasswordFile string
}

func (c *RedisConfig) SetFlags(fs *flag.FlagSet) {
	fs.Var(c, "redis.url", "redis connection url (i.e. redis://<user>:<password>@<host>:<port>/<db_number>)")
	fs.BoolVar(&c.Enabled, "redis.enabled", false, "enable use of a redis server")
	fs.StringVar(&c.Addr, "redis.addr", "localhost:6379", "redis address")
	fs.StringVar(&c.Username, "redis.user", "", "redis username")
	fs.StringVar(&c.Password, "redis.password", "", "redis password")
	fs.StringVar(&c.PasswordFile, "redis.passwordFile", "", "path to file containing redis password")
	fs.IntVar(&c.DB, "redis.database", 0, "redis database number")
}

func (c *RedisConfig) Init() error {
	if c.Password != "" && c.PasswordFile != "" {
		return errors.New("redis.password and redis.passwordFile cannot both be set")
	}
	if c.PasswordFile != "" {
		b, err := ioutil.ReadFile(c.PasswordFile)
		if err != nil {
			return nil
		}
		c.Password = string(b)
	}
	return nil
}

func (c *RedisConfig) Set(s string) error {
	opt, err := redis.ParseURL(s)
	if err == nil {
		c.Options = *opt
	}
	return err
}

// redis://<user>:<password>@<host>:<port>/<db_number>

func (c RedisConfig) String() string {
	if c.IsEmpty() {
		return ""
	}
	str := "redis://"
	if c.Username != "" {
		str += c.Username
		if c.Password != "" {
			str += ":" + c.Password
		}
		str += "@"
	}
	str += c.Addr
	str += fmt.Sprintf("/%d", c.DB)
	return str
}

// StringRedacted is the same as string except password and a username are
// redacted. This should be used in logs.
func (c RedisConfig) StringRedacted() string {
	c.Password = strings.Repeat("*", 5)
	c.Username = strings.Repeat("*", 5)
	return c.String()
}

func (c RedisConfig) IsEmpty() bool {
	return c.Addr == "" && c.Username == "" && c.Password == ""
}
