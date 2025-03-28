package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/vredens/go-webservice"
	"gitlab.com/vredens/go-logger/v2"
)

type Recoverer struct {
	Fn    func()
	Stack []string
	Err   error
	done  chan struct{}
}

func NewRecoverer() *Recoverer {
	output := &Recoverer{
		done: make(chan struct{}),
	}
	output.Fn = func() {
		if d := recover(); d != nil {
			if err, ok := d.(error); ok {
				output.Err = fmt.Errorf("PANIC; %w", err)
			} else {
				output.Err = fmt.Errorf("PANIC: %+v", d)
			}
		}
		close(output.done)
	}
	return output
}

func (rec *Recoverer) Wait(timeout time.Duration) error {
	select {
	case <-rec.done:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout")
	}
}

func main() {
	log := slog.New(logger.NewSLogHandler(logger.WithTags("http"), slog.LevelInfo))

	// webservice.

	srv := webservice.NewServer(":8080", webservice.ServerOptions{
		Logger:             log,
		AccessLogDiscarder: webservice.NewAccessLogDiscarder(webservice.AccessLogLevelInfo, regexp.MustCompile("/_/")),
		MetricsMiddleware: webservice.NewMetricsMiddleware(func(method, route, status string, elapsed time.Duration) {
			// TODO: do something with the variables above.
		}),
	})
	srv.RegisterDebugRoutes("_")
	srv.RegisterAdminRoutes("_")
	srv.RegisterHealthRoutes("_")
	srv.Echo.GET("/panic", func(ctx webservice.Context) error {
		panic("my panic room")
	})
	srv.Echo.GET("/", func(ctx webservice.Context) error {
		return webservice.NewError(404, errors.New("poopi"))
	})
	srv.Echo.GET("/nestedpanic", func(ctx webservice.Context) error {
		rec := NewRecoverer()
		go func() {
			defer rec.Fn()
			panic("my panic room")
		}()

		rec.Wait(1 * time.Second)

		if rec.Err != nil {
			return ctx.JSON(http.StatusInsufficientStorage, rec.Err.Error())
		}

		return nil
	})

	srv.Start()
}
