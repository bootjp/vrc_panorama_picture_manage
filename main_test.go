package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gomodule/redigo/redis"
)

func TestPanoramaAPI(t *testing.T) {
	store := newTestRedis()
	originalConnection := redisConnection
	redisConnection = func() (redis.Conn, error) { return store, nil }
	t.Cleanup(func() { redisConnection = originalConnection })
	t.Setenv("REDIS_KEY_PREFIX", "panorama:")
	router := newRouter()

	request := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	const contentURL = "https://example.com/panorama.jpg"
	resp := request(http.MethodPut, "/api/photo", `{"url":"`+contentURL+`"}`, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("temporary token PUT: status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if got := store.value("panorama:photo"); got != contentURL {
		t.Errorf("stored URL = %q, want %q", got, contentURL)
	}

	resp = request(http.MethodGet, "/v1/photo", "", "")
	if resp.Code != http.StatusFound || resp.Header().Get("Location") != contentURL {
		t.Errorf("panorama redirect: status = %d, location = %q", resp.Code, resp.Header().Get("Location"))
	}
	if got := resp.Header().Get("Cache-Control"); got != "max-age=0, s-maxage=1800" {
		t.Errorf("cache control = %q", got)
	}

	resp = request(http.MethodGet, "/api/keys", "", "")
	var keys []string
	if resp.Code != http.StatusOK || json.Unmarshal(resp.Body.Bytes(), &keys) != nil || len(keys) != 1 || keys[0] != "photo" {
		t.Errorf("keys response: status = %d, body = %s", resp.Code, resp.Body.String())
	}

	for _, path := range []string{"/v1/missing", "/v2/missing"} {
		resp = request(http.MethodGet, path, "", "")
		if resp.Code != http.StatusNoContent {
			t.Errorf("%s: status = %d, want 204", path, resp.Code)
		}
	}

	resp = request(http.MethodPut, "/api/rejected", `{"url":"https://example.com/no.jpg"}`, "invalid")
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: status = %d, want 401", resp.Code)
	}
	if got := store.value("panorama:rejected"); got != "" {
		t.Errorf("invalid token wrote URL %q", got)
	}

	resp = request(http.MethodPut, "/api/bad-json", `{"url":`, token)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: status = %d, want 400", resp.Code)
	}
	if got := store.value("panorama:bad-json"); got != "" {
		t.Errorf("malformed JSON wrote URL %q", got)
	}

	store.addSetMember("panorama:tokens", "persistent-token")
	resp = request(http.MethodPut, "/api/persistent", `{"url":"`+contentURL+`"}`, "persistent-token")
	if resp.Code != http.StatusOK {
		t.Errorf("persistent token PUT: status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

// testRedis implements the commands used by the handlers without a Redis server.
type testRedis struct {
	mu     sync.Mutex
	values map[string]string
	sets   map[string]map[string]struct{}
}

func newTestRedis() *testRedis {
	return &testRedis{values: make(map[string]string), sets: make(map[string]map[string]struct{})}
}

func (s *testRedis) value(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}

func (s *testRedis) addSetMember(key, member string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sets[key] == nil {
		s.sets[key] = make(map[string]struct{})
	}
	s.sets[key][member] = struct{}{}
}

func (s *testRedis) Do(command string, args ...interface{}) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := args[0].(string)
	switch command {
	case "GET":
		value, ok := s.values[key]
		if !ok {
			return nil, nil
		}
		return []byte(value), nil
	case "SET":
		s.values[key] = args[1].(string)
		return "OK", nil
	case "SADD":
		if s.sets[key] == nil {
			s.sets[key] = make(map[string]struct{})
		}
		s.sets[key][args[1].(string)] = struct{}{}
		return int64(1), nil
	case "SMEMBERS":
		members := make([]string, 0, len(s.sets[key]))
		for member := range s.sets[key] {
			members = append(members, member)
		}
		sort.Strings(members)
		reply := make([]interface{}, len(members))
		for i, member := range members {
			reply[i] = []byte(member)
		}
		return reply, nil
	default:
		return nil, errors.New("unsupported test command: " + command)
	}
}

func (*testRedis) Close() error                      { return nil }
func (*testRedis) Err() error                        { return nil }
func (*testRedis) Send(string, ...interface{}) error { return errors.New("unsupported test command") }
func (*testRedis) Flush() error                      { return errors.New("unsupported test command") }
func (*testRedis) Receive() (interface{}, error)     { return nil, errors.New("unsupported test command") }
