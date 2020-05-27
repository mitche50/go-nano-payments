package main

import (
	"fmt"
	"testing"

	"github.com/gomodule/redigo/redis"
)

func TestHello(t *testing.T) {
	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("There was an error: ", err)
	}
	c.Close()
	//got := Hello(c)
	//want := "world!"

	//if got != want {
	//	t.Errorf("got '%s' want '%s'", got, want)
	//}
}
