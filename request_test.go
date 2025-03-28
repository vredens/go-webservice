package webservice

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestCreation(t *testing.T) {
	cli01 := NewClient("http://localhost:80")
	req01 := cli01.NewRequest()

	assert.NotNil(t, req01, "request is not empty")
}

func TestRequestWithMiddlewares(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the request
		switch r.URL.Path {
		case "/t01":
			assert.Equal(t, "v1", r.Header.Get("h1"))
			assert.Equal(t, "v1", r.Header.Get("h1"))
		case "/t02":
			assert.Equal(t, "v2", r.Header.Get("h1"))
			t.Logf("%+v", r.Header)
		default:
			http.Error(w, "nok", http.StatusTeapot)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	t.Run("ok", func(t *testing.T) {
		cli := NewClient(srv.URL)
		req := cli.NewRequest()
		s, p, err := req.WithHeader("h1", "v1").Do(context.TODO(), http.MethodGet, "/t01", nil)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, s)
		assert.Equal(t, []byte("ok"), p)
		s, p, err = req.WithUniqueHeader("h1", "v2").Do(context.TODO(), http.MethodGet, "/t02", nil)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, s)
		assert.Equal(t, []byte("ok"), p)
	})

	t.Run("error", func(t *testing.T) {
		cli := NewCustomClient(srv.URL, ClientOptions{
			Middlewares: []RequestMiddleware{
				func(ctx context.Context, req *http.Request) (*http.Request, error) {
					return req, fmt.Errorf("puff")
				},
			},
		})
		_, _, err := cli.Request(context.TODO(), http.MethodGet, "/t02", nil)
		assert.Error(t, err)
	})
}
