package webservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gitlab.com/vredens/go-logger/v2"
)

// Context is a server Request/Response context.
// Type alias to echo's Context.
type Context = echo.Context

type ServerOptions struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	TLSCertFile       string
	TLSKeyFile        string
	// Logger for internal messages and errors.
	Logger *slog.Logger
	// AccessLogDisabled will not log any access logs if set to true.
	AccessLogDisabled bool
	// AccessLogDiscarder function should return true when no access log is to be written.
	AccessLogDiscarder func(c Context) bool
	// AccessLogMiddleware will override any AccessLog configuration if set.
	// This is the second-last middleware called.
	AccessLogMiddleware echo.MiddlewareFunc
	// MetricsMiddleware will be the last middleware called.
	MetricsMiddleware echo.MiddlewareFunc
	GzipDisabled      bool
	GzipSkipper       func(c Context) bool
}

// Server is a wrapper around echo.Echo.
type Server struct {
	Echo    *echo.Echo
	log     logger.SLogger
	address string
	running uint32
	tls     struct {
		enabled  int32
		certFile string
		keyFile  string
	}
}

// NewServer ...
func NewServer(address string, opts ServerOptions) *Server {
	srv := &Server{
		address: address,
	}

	if opts.Logger != nil {
		srv.log = logger.NewSLogWrapper(opts.Logger)
	} else {
		srv.log = logger.NewSLogWrapper(slog.Default()).WithTags("http")
	}

	srv.Echo = echo.New()
	srv.Echo.HideBanner = true
	srv.Echo.HidePort = true
	srv.Echo.HTTPErrorHandler = srv.webErrorHandler
	srv.Echo.Server.ReadHeaderTimeout = opts.ReadHeaderTimeout
	srv.Echo.Server.ReadTimeout = opts.ReadTimeout
	srv.Echo.Server.WriteTimeout = opts.WriteTimeout
	srv.Echo.Server.IdleTimeout = opts.IdleTimeout
	srv.Echo.TLSServer.ReadHeaderTimeout = opts.ReadHeaderTimeout
	srv.Echo.TLSServer.ReadTimeout = opts.ReadTimeout
	srv.Echo.TLSServer.WriteTimeout = opts.WriteTimeout
	srv.Echo.TLSServer.IdleTimeout = opts.IdleTimeout

	if opts.TLSCertFile != "" {
		srv.tls.certFile = opts.TLSCertFile
		srv.tls.keyFile = opts.TLSKeyFile
		srv.tls.enabled = 1
	}

	if opts.MetricsMiddleware != nil {
		srv.Echo.Use(opts.MetricsMiddleware)
	}
	if opts.AccessLogMiddleware != nil {
		srv.Echo.Use(opts.AccessLogMiddleware)
	} else if !opts.AccessLogDisabled {
		srv.Echo.Use(NewAccessLogMiddleware(AccessLogger{
			Logger:    srv.log.Logger,
			Discarder: opts.AccessLogDiscarder,
		}))
	}
	srv.Echo.Use(srv.recoverMiddleware())

	if !opts.GzipDisabled {
		srv.Echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{
			Skipper: opts.GzipSkipper,
		}))
	}

	return srv
}

// RegisterAdminRoutes registers preset handlers for <prefix>/admin routes.
func (srv *Server) RegisterAdminRoutes(prefix string) {
	srv.Echo.POST(prefix+"/admin/shutdown", srv.handleShutdown)
}

func (srv *Server) handleShutdown(c Context) error {
	go srv.Stop()

	return c.NoContent(http.StatusNoContent)
}

// RegisterHealthRoutes registers preset handlers for <prefix>/health and <prefix>/info routes.
func (srv *Server) RegisterHealthRoutes(prefix string) {
	srv.Echo.GET(prefix+"/health", srv.handleGetApplicationQuickStatus)

	// TODO: use debug.ReadBuildInfo()

	var tryPaths = []string{
		"/etc/build.properties",
		"./build.properties",
	}
	for _, path := range tryPaths {
		if _, err := os.Stat(path); err == nil {
			srv.Echo.File(prefix+"/info", path)
			break
		}
	}
}

func (srv *Server) handleGetApplicationQuickStatus(context Context) error {
	return context.JSON(http.StatusOK, nil)
}

// Start launches the HTTP Server and writes the exit
func (srv *Server) Start() error {
	if !atomic.CompareAndSwapUint32(&srv.running, 0, 1) {
		return fmt.Errorf("server is not in pre-running state")
	}

	srv.log.Infof("webserver: starting [address:%s]", srv.address)
	err := srv.start()
	srv.log.Infof("webserver: shutting down [address:%s]", srv.address)

	atomic.StoreUint32(&srv.running, 0)

	if err.Error() == "http: Server closed" {
		return nil
	}

	return err
}

func (srv *Server) start() error {
	if atomic.LoadInt32(&srv.tls.enabled) == 1 {
		return srv.Echo.StartTLS(srv.address, srv.tls.certFile, srv.tls.keyFile)
	}

	return srv.Echo.Start(srv.address)
}

// Stop performs a clean shutdown of the server.
func (srv *Server) Stop() error {
	if atomic.LoadUint32(&srv.running) != 1 {
		return nil
	}
	return srv.Echo.Server.Shutdown(context.Background())
}

func (srv *Server) webErrorHandler(err error, c Context) {
	if c.Response().Committed {
		return
	}

	var (
		code = http.StatusInternalServerError
		msg  string
	)
	if e, ok := err.(*echo.HTTPError); ok {
		code = e.Code
		msg = fmt.Sprintf(`{"message":%q}`, e.Message)
	} else if err, ok := err.(Error); ok {
		code = err.Code
		if code < 400 || code >= 600 {
			code = http.StatusInternalServerError
		}
		msg = err.JSONFormatter()
	} else {
		msg = fmt.Sprintf(`{"message":%q}`, http.StatusText(http.StatusInternalServerError))
	}

	if c.Request().Method == echo.HEAD {
		if err := c.NoContent(code); err != nil {
			srv.log.Errorf("error sending response to client: %+v", err)
		}
		return
	}

	if err := c.JSONBlob(code, []byte(msg)); err != nil {
		srv.log.Errorf("error sending response to client: %+v", err)
	}
}

func (srv Server) recoverMiddleware() echo.MiddlewareFunc {
	var config middleware.RecoverConfig

	if config.Skipper == nil {
		config.Skipper = middleware.DefaultRecoverConfig.Skipper
	}
	if config.StackSize == 0 {
		config.StackSize = middleware.DefaultRecoverConfig.StackSize
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, true)
					srv.log.WithTags("ALERT").With(slog.String("strace", string(stack[:length]))).Errorf("[PANIC RECOVER] %+v", err)
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}
