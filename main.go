package main

import (
	"embed"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed index.html
var staticFiles embed.FS

var logger = log.New(os.Stdout, "vrc_ppm: ", log.LstdFlags)
var token = uuid.Must(uuid.NewRandom()).String()

func main() {
	logger.Printf("current temporary token %s \n", token)

	go Ticker()

	e := echo.New()

	// Routes
	e.GET("/_/", echo.WrapHandler(http.StripPrefix("/_/", http.FileServer(http.FS(staticFiles)))))
	e.GET("/v1/:key", panoramaHandler)
	e.GET("/v2/:key", mp4Handler)
	e.GET("/api/keys", keysHandler)
	g := e.Group("/api/:key", middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		return validToken(key)
	}))
	g.PUT("", putHandler)

	e.Logger.Fatal(e.Start(":1323"))
}

func Ticker() {
	for range time.Tick(time.Minute) {
		r, err := redisConnection()
		if err != nil {
			logger.Println(err)
			continue
		}

		k, err := redis.Strings(r.Do("SMEMBERS", "keys"))
		if err != nil {
			logger.Println(err)
			continue
		}
		for _, key := range k {
			url, err := getContentURLByKey(key)
			if err != nil {
				logger.Println(err)
				continue
			}
			if ok, _ := checkCacheExists(url); ok {
				continue
			}

			data, err := fetchContentByURL(url)
			if err != nil {
				logger.Println(err)
				continue
			}

			image, err := generateMP4(data)
			if err != nil {
				logger.Println(err)
				continue
			}

			err = cachePut(url, image)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("cache generate", url)
		}
		err = r.Close()
		if err != nil {
			logger.Println(err)
			continue
		}
	}
}

// panoramaHandler is response redirect endpoint
// in case of follow redirect client
// ex) VRChat SDK2 World Panorama Component
func panoramaHandler(c echo.Context) error {
	key := c.Param("key")
	c.Response().Header().Set("Cache-Control", "max-age=0, s-maxage=1800")
	c.Response().Header().Set("CDN-Cache-Control", "maxage=1800")

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
	c.Response().Header().Set("Cache-Control", "max-age=0, s-maxage=1800")
	c.Response().Header().Set("CDN-Cache-Control", "maxage=1800")

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

	cacheExists, movie := checkCacheExists(url)
	if cacheExists {
		return c.Blob(200, "video/mp4", movie)
	}

	movie, err = generateMP4(data)
	if err != nil {
		logger.Println(err)
	}

	err = cachePut(url, movie)
	if err != nil {
		logger.Println(err)
	}

	return c.Blob(200, "video/mp4", movie)
}

func cachePut(url string, movie []byte) error {
	h := hash(url)
	return os.WriteFile(os.TempDir()+strconv.Itoa(int(h)), movie, 0644)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func checkCacheExists(url string) (bool, []byte) {
	h := hash(url)
	f, err := os.Open(os.TempDir() + strconv.Itoa(int(h)))
	if err != nil {
		return false, nil
	}

	d, err := io.ReadAll(f)
	if err != nil {
		return false, nil
	}

	return true, d
}

type (
	PutRequest struct {
		URL string `json:"url"`
	}
)

// putHandler is handling manage request
// check temporary token or in redis persistent token
func putHandler(c echo.Context) error {
	key := c.Param("key")

	u := &PutRequest{}
	if err := c.Bind(u); err != nil {
		return c.String(400, `{"message": "invalid request"}`)
	}

	r, err := redisConnection()
	if err != nil {
		logger.Println(err)
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Println(err)
		}
	}()

	_, err = r.Do("SET", key, u.URL)
	if err != nil {
		return c.String(500, `{"message": "data save failed"}`)
	}
	if err != nil {
		return c.String(500, `{"message": "data save failed"}`)
	}
	_, err = r.Do("SADD", "keys", key)
	if err != nil {
		return c.String(500, `{"message": "data save failed"}`)
	}

	return c.String(200, `{"message":"success"}`)
}

func keysHandler(c echo.Context) error {
	r, err := redisConnection()
	if err != nil {
		logger.Println(err)
		return c.JSON(500, `{"message": "data save failed"}`)
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Println(err)
		}
	}()

	k, err := redis.Strings(r.Do("SMEMBERS", "keys"))
	if err != nil {
		logger.Println(err)
		return c.JSON(500, `{"message": "data save failed"}`)
	}

	return c.JSON(200, k)
}

func validToken(req string) (bool, error) {
	if token == req {
		return true, nil
	}
	r, err := redisConnection()
	if err != nil {
		return false, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Print(err)
		}
	}()

	tokens, err := redis.Strings(r.Do("SMEMBERS", "tokens"))
	if err != nil {
		return false, err
	}

	for _, v := range tokens {
		if v == req {
			return true, nil
		}
	}

	return false, nil
}

func getContentURLByKey(key string) (string, error) {
	r, err := redisConnection()
	if err != nil {
		logger.Println(err)
		return "", err
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Println(err)
		}

	}()
	return redis.String(r.Do("GET", key))
}

func fetchContentByURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return io.ReadAll(resp.Body)
}

func generateMP4(data []byte) ([]byte, error) {

	imgFile, err := os.CreateTemp(os.TempDir(), "vrc_ppm")
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	movFile, err := os.CreateTemp(os.TempDir(), "vrc_ppm")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(imgFile.Name())
		_ = os.RemoveAll(movFile.Name())
	}()

	err = os.Rename(movFile.Name(), movFile.Name()+".mp4")
	if err != nil {
		return nil, err
	}
	movFile, err = os.Open(movFile.Name() + ".mp4")
	if err != nil {
		return nil, err
	}

	_, err = imgFile.Write(data)
	if err != nil {
		logger.Println(err)
		return nil, err
	}

	err = ffmpeg.Input(imgFile.Name(), ffmpeg.KwArgs{"loop": 1}).
		Output(movFile.Name(),
			ffmpeg.KwArgs{"t": 1},
			ffmpeg.KwArgs{"vcodec": "libx264"},
			ffmpeg.KwArgs{"profile:v": "baseline"},
			ffmpeg.KwArgs{"pix_fmt": "yuv420p"},
			ffmpeg.KwArgs{"format": "mp4"},
		).OverWriteOutput().WithTimeout(2 * time.Minute).Run()
	if err != nil {
		return nil, err
	}

	return io.ReadAll(movFile)
}

func redisConnection() (redis.Conn, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	// fill default port
	if !strings.Contains(host, ":") {
		host += ":6379"
	}

	var opts []redis.DialOption
	if pw := os.Getenv("REDIS_PASSWORD"); pw != "" {
		opts = append(opts, redis.DialPassword(pw))
	}

	c, err := redis.Dial("tcp", host, opts...)
	if err != nil {
		return nil, err
	}

	return c, nil
}
