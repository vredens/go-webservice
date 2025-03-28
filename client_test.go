package webservice

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	var o1 = ClientOptions{}
	var o2 = ClientOptions{MaxRequestTimeout: 10 * time.Second}

	o1 = o1.sanitize()
	assert.NotNil(t, o1.Conn, "original connection set after sanitize")
	assert.NotZero(t, o1.MaxRequestTimeout, "original timeout set after sanitize")

	_ = NewCustomClient("http://127.0.0.1:8080", o2)
	assert.Equal(t, 10*time.Second, o2.MaxRequestTimeout, "client conn timeout not properly set")

	o1.MaxRequestTimeout = 50 * time.Second
	cli := NewCustomClient("http://127.0.0.1:8080", o1)
	assert.Equal(t, 50*time.Second, cli.conn.Timeout, "client conn timeout not properly set")
}

func TestClient_NewRequest(t *testing.T) {
	t.Run("client with middlewares", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{
			Middlewares: []RequestMiddleware{
				func(ctx context.Context, req *http.Request) (*http.Request, error) {
					req.Header.Add("mh", "mv")
					return req, nil
				},
			},
		})
		req, err := cli.NewRequest().Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, "mv", req.Header.Get("mh"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 2, len(req.Header))
	})

	t.Run("simple request", func(t *testing.T) {
		cli := NewClient("http://127.0.0.1:8080")
		req, err := cli.NewRequest(
			cli.RequestTimeout(5*time.Second),
			cli.RequestHeader("h1", "v1"),
			cli.RequestHeader("h1", "v2"),
			cli.RequestUniqueHeader("h2", "v1"),
			cli.RequestUniqueHeader("h2", "v2"),
		).Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 3, len(req.Header))
	})

	t.Run("simple request", func(t *testing.T) {
		cli := NewClient("http://127.0.0.1:8080")
		req, err := cli.NewRequest().
			WithTimeout(5*time.Second).
			WithHeader("h1", "v1").
			WithHeader("h1", "v2").
			WithUniqueHeader("h2", "v1").
			WithUniqueHeader("h2", "v2").
			Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 3, len(req.Header))
	})

	t.Run("default headers", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{}.AddHeaders(map[string]string{"dh": "z"}))
		req, err := cli.NewRequest(
			cli.RequestTimeout(5*time.Second),
			cli.RequestHeader("h1", "v1"),
			cli.RequestHeader("h1", "v2"),
			cli.RequestUniqueHeader("h2", "v1"),
			cli.RequestUniqueHeader("h2", "v2"),
		).Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, req.Header.Values("dh"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 4, len(req.Header))
	})

	t.Run("default headers", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{}.AddHeaders(map[string]string{"dh": "z"}))
		req, err := cli.NewRequest().
			WithTimeout(5*time.Second).
			WithHeader("h1", "v1").
			WithHeader("h1", "v2").
			WithUniqueHeader("h2", "v1").
			WithUniqueHeader("h2", "v2").
			Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, req.Header.Values("dh"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 4, len(req.Header))
	})

	t.Run("json request", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{}.AddHeaders(map[string]string{"dh": "z"}))
		req, err := cli.NewJSONRequest(
			cli.RequestTimeout(5*time.Second),
			cli.RequestHeader("h1", "v1"),
			cli.RequestHeader("h1", "v2"),
			cli.RequestUniqueHeader("h2", "v1"),
			cli.RequestUniqueHeader("h2", "v2"),
		).Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, req.Header.Values("dh"))
		assert.Equal(t, []string{"application/json"}, req.Header.Values("content-type"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 5, len(req.Header))
	})

	t.Run("json request", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{}.AddHeaders(map[string]string{"dh": "z"}))
		req, err := cli.NewJSONRequest().
			WithTimeout(5*time.Second).
			WithHeader("h1", "v1").
			WithHeader("h1", "v2").
			WithUniqueHeader("h2", "v1").
			WithUniqueHeader("h2", "v2").
			Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"v1", "v2"}, req.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, req.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, req.Header.Values("dh"))
		assert.Equal(t, []string{"application/json"}, req.Header.Values("content-type"))
		assert.NotZero(t, req.Header.Values("user-agent"))
		assert.Equal(t, 5, len(req.Header))
	})

	t.Run("json requests", func(t *testing.T) {
		cli := NewCustomClient("http://127.0.0.1:8080", ClientOptions{}.AddHeaders(map[string]string{"dh": "z"}))
		req := cli.NewJSONRequest()
		r1 := req.WithTimeout(5*time.Second).WithHeader("h1", "v1")
		r2 := r1.WithTimeout(time.Second).WithHeader("h1", "v2").WithHeader("h2", "v1")
		r3 := r2.WithUniqueHeader("h2", "v2")

		h1, err := r1.Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		h2, err := r2.Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)
		h3, err := r3.Prepare(context.Background(), "GET", "/", nil)
		assert.NoError(t, err)

		assert.Equal(t, 5*time.Second, r1.core.core.timeout)
		assert.Equal(t, []string{"v1"}, h1.Header.Values("h1"))
		assert.Equal(t, []string{"z"}, h1.Header.Values("dh"))
		assert.Equal(t, []string{"application/json"}, h1.Header.Values("content-type"))
		assert.NotZero(t, h1.Header.Values("user-agent"))
		assert.Equal(t, 4, len(h1.Header))

		assert.Equal(t, time.Second, r2.core.core.timeout)
		assert.Equal(t, []string{"v1", "v2"}, h2.Header.Values("h1"))
		assert.Equal(t, []string{"v1"}, h2.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, h2.Header.Values("dh"))
		assert.Equal(t, []string{"application/json"}, h2.Header.Values("content-type"))
		assert.NotZero(t, h2.Header.Values("user-agent"))
		assert.Equal(t, 5, len(h2.Header))

		assert.Equal(t, time.Second, r3.core.core.timeout)
		assert.Equal(t, []string{"v1", "v2"}, h3.Header.Values("h1"))
		assert.Equal(t, []string{"v2"}, h3.Header.Values("h2"))
		assert.Equal(t, []string{"z"}, h3.Header.Values("dh"))
		assert.Equal(t, []string{"application/json"}, h3.Header.Values("content-type"))
		assert.NotZero(t, h3.Header.Values("user-agent"))
		assert.Equal(t, 5, len(h3.Header))
	})
}
