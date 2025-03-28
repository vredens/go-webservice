package webservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vredens/go-webservice"
	"gitlab.com/vredens/go-logger/v2"
)

func init() {
	logger.Reconfigure(logger.ConfigWriter(io.Discard))
}

func serverStart(srv *webservice.Server) <-chan error {
	var done = make(chan error)

	go func() {
		done <- srv.Start()
	}()

	// lame way to wait for the server to start
	<-time.After(10 * time.Millisecond)

	return done
}

func serverStop(srv *webservice.Server) <-chan error {
	var done = make(chan error)

	go func() {
		done <- srv.Stop()
	}()

	return done
}

func waitOnChan(done <-chan error) error {
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}

var hugePayloadString string

func hugePayloadHandler(ctx webservice.Context) error {
	return ctx.JSON(200, struct {
		PL string `json:"payload"`
	}{
		PL: hugePayloadString,
	})
}

func newServer(opts webservice.ServerOptions) *webservice.Server {
	var srv = webservice.NewServer(":8001", opts)
	srv.RegisterHealthRoutes("/_")
	srv.Echo.GET("/huge", hugePayloadHandler)

	return srv
}

func BenchmarkServer(b *testing.B) {
	var cli = webservice.NewClient("http://127.0.0.1:8001")
	for i := 0; i < 1e5; i++ {
		hugePayloadString += "huge payload is "
	}
	var hugePayloadSize = len([]byte(hugePayloadString)) + 14 // {"payload":""} == 14 chars

	b.Run("404/nogzip", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/notfound", nil)
				assert.Nil(b, err)
				assert.Equal(b, 404, s)
				assert.NotNil(b, res)
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})

	b.Run("404/gzip", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/notfound", nil)
				assert.Nil(b, err)
				assert.Equal(b, 404, s)
				assert.NotNil(b, res)
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})

	b.Run("ping/nogzip", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{GzipDisabled: true})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/_/health", nil)
				assert.Nil(b, err)
				assert.Equal(b, 200, s)
				assert.NotNil(b, res)
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})

	b.Run("ping/gzip", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/_/health", nil)
				assert.Nil(b, err)
				assert.Equal(b, 200, s)
				assert.NotNil(b, res)
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})

	b.Run("nogzip/huge", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{GzipDisabled: true})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/huge", nil)
				assert.Nil(b, err)
				assert.Equal(b, 200, s)
				assert.NotNil(b, res)
				assert.Equal(b, hugePayloadSize+1, len(res))
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})
	b.Run("gzip/huge", func(b *testing.B) {
		var srv = newServer(webservice.ServerOptions{})
		var doneStart = serverStart(srv)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/huge", nil)
				assert.Nil(b, err)
				assert.Equal(b, 200, s)
				assert.NotNil(b, res)
				assert.Equal(b, hugePayloadSize+1, len(res))
			}
		})
		b.StopTimer()
		var doneStop = serverStop(srv)
		assert.Nil(b, waitOnChan(doneStart), "failed to start server")
		assert.Nil(b, waitOnChan(doneStop), "failed to stop server")
	})
}

func TestServerNoRoute(t *testing.T) {
	var srv = webservice.NewServer("127.0.0.1:8001", webservice.ServerOptions{})
	var doneStart = serverStart(srv)

	var cli = webservice.NewClient("http://127.0.0.1:8001")
	var s, res, err = cli.Request(context.TODO(), http.MethodGet, "/", nil)
	assert.Nil(t, err)
	assert.Equal(t, 404, s)
	assert.NotNil(t, res)
	assert.Equal(t, `{"message":"Not Found"}`, string(res))

	var doneStop = serverStop(srv)
	assert.Nil(t, waitOnChan(doneStart), "failed to start server")
	assert.Nil(t, waitOnChan(doneStop), "failed to stop server")
}

func TestServerAccessLogs(t *testing.T) {
	b := &bytes.Buffer{}
	w := logger.New(logger.ConfigWriter(b))
	var log = slog.New(logger.NewSLogHandler(w.Spawn(), slog.LevelDebug))

	var srv = webservice.NewServer("127.0.0.1:8001", webservice.ServerOptions{Logger: log})

	srv.Echo.GET("/sample/:test", func(ctx webservice.Context) error {
		switch ctx.Param("test") {
		case "one":
			ctx.Request().Header.Add("X-Tags", "tag1")
			ctx.Request().Header.Add("X-Tags", "tag2")
		case "two":
		}
		return ctx.NoContent(200)
	})

	var doneStart = serverStart(srv)

	var cli = webservice.NewClient("http://127.0.0.1:8001")

	var reqHeaders = map[string]string{
		"X-Request-ID": "ID",
	}
	var status, _, err = cli.NewRequest().WithHeaders(reqHeaders).WithTimeout(time.Second).Do(context.TODO(), "GET", "/sample/one", nil)
	assert.Nil(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, b.String(), `"tag1","tag2"`)
	b.Reset()
	status, _, err = cli.NewRequest().WithHeaders(reqHeaders).WithTimeout(time.Second).Do(context.TODO(), "GET", "/sample/two", nil)
	assert.Nil(t, err)
	assert.Equal(t, 200, status)
	assert.NotContains(t, b.String(), `"tag1","tag2"`)
	b.Reset()

	var doneStop = serverStop(srv)
	assert.Nil(t, waitOnChan(doneStart), "failed to terminate server")
	assert.Nil(t, waitOnChan(doneStop), "failed to stop server")
}

func TestServerPanicRecover(t *testing.T) {
	b := &bytes.Buffer{}
	w := logger.New(logger.ConfigWriter(b))
	var log = slog.New(logger.NewSLogHandler(w.Spawn(), slog.LevelDebug))

	var srv = webservice.NewServer("127.0.0.1:8001", webservice.ServerOptions{Logger: log})
	srv.Echo.GET("/sample/:test", func(ctx webservice.Context) error {
		panic("oops")
	})

	var doneStart = serverStart(srv)

	var cli = webservice.NewClient("http://127.0.0.1:8001")

	var status, _, err = cli.NewRequest().WithTimeout(time.Second).Do(context.TODO(), "GET", "/sample/one", nil)
	assert.Nil(t, err)
	assert.Equal(t, 500, status)
	assert.Contains(t, b.String(), "[PANIC RECOVER] oops")

	var doneStop = serverStop(srv)
	assert.Nil(t, waitOnChan(doneStart), "failed to terminate server")
	assert.Nil(t, waitOnChan(doneStop), "failed to stop server")
}
