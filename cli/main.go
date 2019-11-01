package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"

	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo"
)

const envTempToken = "TEMPORARY_TOKEN"

func main() {
	temporaryToken := uuid.Must(uuid.NewRandom())
	err := os.Setenv(envTempToken, temporaryToken.String())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("current temporary token %s \n", temporaryToken)
	e := echo.New()

	// Routes
	e.GET("/r/:key", panoramaHandler)
	e.POST("/api/update", apiHandler)
	e.File("/r/__test__", "public/index.html")

	e.Logger.Fatal(e.Start(":1323"))
}

func panoramaHandler(c echo.Context) error {
	key := c.Param("key")
	r := redisConnection()
	s, err := redis.String(r.Do("GET", key))
	if err != nil {
		// redirect resource not found.
		log.Println(err)
		return c.NoContent(204)
	}
	err = r.Close()
	if err != nil {
		log.Println(err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Redirect(302, s)
}

type (
	UpdateRequest struct {
		Token string
		Key   string
		URL   string
	}
)

func apiHandler(c echo.Context) error {
	u := &UpdateRequest{}
	if err := c.Bind(u); err != nil {
		return err
	}
	if !ValidToken(u.Token) {
		c.Response().Status = 400
		return nil
	}
	r := redisConnection()
	_, err := r.Do("SET", u.Key, u.URL)
	return err
}

func ValidToken(token string) bool {
	tt := os.Getenv(envTempToken)
	if tt != "" && tt == token {
		return true
	}
	r := redisConnection()

	tokens, err := redis.Strings(r.Do("SMEMBERS", "tokens"))
	if err != nil {
		return false
	}
	err = r.Close()
	if err != nil {
		log.Println(err)
	}
	for _, v := range tokens {
		if v == token {
			return true
		}
	}

	return false
}

func redisConnection() redis.Conn {
	host := os.Getenv("REDIS_HOST")

	c, err := redis.Dial("tcp", host+":6379")
	if err != nil {
		log.Println(err)
	}
	return c
}
