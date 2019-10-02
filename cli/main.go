package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/go/src/os"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", handler)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}


func handler(c echo.Context) error {
	k :=c.Path()
	r := redisConnection()
	s, err := redis.String(r.Do("GET", k))
	if err != nil {
		fmt.Println(err)
	}

	return c.Redirect(302, s)
}

func redisConnection() redis.Conn {
	host := os.Getenv("REDIS_HOST")

	c, err := redis.Dial("tcp", host + ":6379")
	if err != nil {
		panic(err)
	}
	return c
}
