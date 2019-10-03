package main

import (
	"log"
	"os"

	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo"
)

func main() {
	e := echo.New()

	// Routes
	// ex) /resource_key
	// ex) /r/resource_key
	e.GET("/:key", handler)
	e.GET("/r/:key", handler)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

func handler(c echo.Context) error {
	key := c.Param("key")
	r := redisConnection()
	s, err := redis.String(r.Do("GET", key))
	if err != nil {
		// redirect resource not found.
		log.Println(err)
		return c.NoContent(204)
	}
	return c.Redirect(302, s)
}

func redisConnection() redis.Conn {
	host := os.Getenv("REDIS_HOST")

	c, err := redis.Dial("tcp", host+":6379")
	if err != nil {
		log.Println(err)
	}
	return c
}
