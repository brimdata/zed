package zqd

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func NewRedisClient(ctx context.Context, conf RedisConfig) (*redis.Client, error) {
	if !conf.Enabled {
		return nil, nil
	}
	client := redis.NewClient(&conf.Options)
	if cmd := client.Ping(ctx); cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return client, nil
}

type RedisConfig struct {
	redis.Options

	Enabled bool
}

func (c *RedisConfig) SetFlags(fs *flag.FlagSet) {
	fs.Var(c, "redis.url", "redis connection url (i.e. redis://<user>:<password>@<host>:<port>/<db_number>)")
	fs.BoolVar(&c.Enabled, "redis.enabled", false, "enable use of a redis server")
	fs.StringVar(&c.Addr, "redis.addr", "localhost:6379", "redis address")
	fs.StringVar(&c.Username, "redis.user", "", "redis username")
	fs.StringVar(&c.Password, "redis.password", "", "redis password")
	fs.IntVar(&c.DB, "redis.database", 0, "redis database number")
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

func (c RedisConfig) IsEmpty() bool {
	return c.Addr == "" && c.Username == "" && c.Password == ""
}
