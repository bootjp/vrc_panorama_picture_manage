package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/bootjp/vrc_panoprama_picture_manage/statik"
	"github.com/garyburd/redigo/redis"
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/rakyll/statik/fs"
)

const envTempToken = "TEMPORARY_TOKEN"

var logger = log.New(os.Stdout, "vrc_panoprama_picture_manage: ", log.LstdFlags)

func main() {
	temporaryToken := uuid.Must(uuid.NewRandom())
	err := os.Setenv(envTempToken, temporaryToken.String())
	if err != nil {
		logger.Fatalln(err)
	}
	logger.Printf("current temporary token %s \n", temporaryToken)
	e := echo.New()

	// Routes
	statikFS, err := fs.New()
	if err != nil {
		logger.Fatalln(err)
	}
	e.GET("/_/", echo.WrapHandler(http.StripPrefix("/_/", http.FileServer(statikFS))))
	e.GET("/v1/:key", panoramaHandler)
	e.GET("/v2/:key", mp4Handler)
	e.POST("/api/update", apiHandler)
	e.Logger.Fatal(e.Start(":1323"))
}

// panoramaHandler is response redirect endpoint
// in case of follow redirect client
// ex) VRChat SDK2 World Panorama Component
func panoramaHandler(c echo.Context) error {
	key := c.Param("key")
	c.Response().Header().Set("Cache-Control", "no-store")

	url, err := getContentURLByKey(key)
	if err != nil {
		// redirect resource not found.
		logger.Println(err)
		return c.NoContent(204)
	}

	return c.Redirect(302, url)
}

// mp4Handler is response 1 sec mp4 movie
// in case of only support movie only client
// ex) VRChat SDK3 World Video Component
func mp4Handler(c echo.Context) error {
	key := c.Param("key")
	c.Response().Header().Set("Cache-Control", "no-store")

	url, err := getContentURLByKey(key)
	if err != nil {
		// redirect resource not found.
		logger.Println(err)
		return c.NoContent(204)
	}

	data, err := fetchContentByURL(url)
	if err != nil {
		logger.Println(err)
		return c.NoContent(204)
	}

	movie, err := generateMP4(data)
	if err != nil {
		logger.Println(err)
	}

	return c.Blob(200, "video/mp4", movie)
}

type (
	UpdateRequest struct {
		Token string
		Key   string
		URL   string
	}
)

// apiHandler is handling manage request
// check temporary token or in redis persistent token
func apiHandler(c echo.Context) error {
	u := &UpdateRequest{}
	if err := c.Bind(u); err != nil {
		return c.String(400, `{"message": "invalid request"}`)
	}
	if !validToken(u.Token) {
		return c.String(400, `{"message": "invalid request"}`)
	}
	r, _ := redisConnection()
	_, err := r.Do("SET", u.Key, u.URL)
	if err != nil {
		return c.String(500, `{"message": "data save failed"}`)
	}
	err = r.Close()
	if err != nil {
		return c.String(500, `{"message": "data save failed"}`)
	}

	return c.String(200, `{"message":"success"}`)
}

func validToken(token string) bool {
	tt := os.Getenv(envTempToken)
	if tt != "" && tt == token {
		return true
	}
	r, err := redisConnection()
	if err != nil {
		log.Println(err)
		return false
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Print(err)
		}
	}()

	tokens, err := redis.Strings(r.Do("SMEMBERS", "tokens"))
	if err != nil {
		return false
	}

	for _, v := range tokens {
		if v == token {
			return true
		}
	}

	return false
}

func getContentURLByKey(key string) (string, error) {
	r, _ := redisConnection()
	defer func() {
		if err := r.Close(); err != nil {
			logger.Println(err)
		}

	}()
	return redis.String(r.Do("GET", key))
}

func fetchContentByURL(url string) ([]byte, error) {
	// todo fetch http content
	return nil, nil
}

func generateMP4(data []byte) ([]byte, error) {
	// todo generate mp4
	return nil, nil
}

func redisConnection() (redis.Conn, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	// fill default port
	if !strings.Contains(":", host) {
		host += ":6379"
	}

	c, err := redis.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	return c, nil
}
