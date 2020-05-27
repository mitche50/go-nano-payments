package nanoredis

import (
	"fmt"
	"os"

	"github.com/gomodule/redigo/redis"
)

//NewPool returns a redis pool for connections using the configured host and port.
func NewPool() *redis.Pool {
	host := os.Getenv("REDISHOST")
	port := os.Getenv("REDISPORT")
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// max number of connections
		MaxActive: 12000,

		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}
