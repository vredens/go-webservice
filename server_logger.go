package webservice

import (
	"log/slog"
	"net/http"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gitlab.com/vredens/go-logger/v2"
)

// AccessLogger settings.
type AccessLogger struct {
	// Logger used for writting access logs.
	// If no logger is passed then no access logs will be written.
	Logger *slog.Logger
	// Discarder func can be used to ignore specific requests from being logged.
	Discarder func(c Context) bool
}

func NewAccessLogMiddleware(params AccessLogger) echo.MiddlewareFunc {
	if params.Logger == nil {
		return nil
	}
	if params.Discarder == nil {
		params.Discarder = func(c Context) bool { return false }
	}
	out := accessLogger{
		discard: params.Discarder,
		alog:    logger.NewSLogWrapper(params.Logger),
	}
	return out.Middleware
}

// accessLogger will use Go's slog.Logger to write access logs.
// You can set a discarder method that will discard certain requests from being logged.
type accessLogger struct {
	alog    logger.SLogger
	discard func(c Context) bool
}

func (logger accessLogger) getRequestID(c Context) string {
	if id := c.Request().Header.Get(echo.HeaderXRequestID); id != "" {
		return id
	}
	if id := c.Response().Header().Get(echo.HeaderXRequestID); id != "" {
		return id
	}
	return ""
}

func (logger accessLogger) sanitizePath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func (logger accessLogger) parseContentLength(cl string) int64 {
	if cl == "" {
		return 0
	}
	n, err := strconv.ParseInt(cl, 0, 64)
	if err != nil {
		return 0
	}
	return n
}

func (logger accessLogger) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c Context) (err error) {
		start := time.Now()

		if err = next(c); err != nil {
			// we are forcing an error handling here so we accurately measure replies even when handlers return an error.
			// this will cause at least one more call to the error handler, since the error is returned, but due to the
			// check of echo.Context.Response().Committed this can be safelly called multiple times.
			c.Error(err)
		}

		elapsed := time.Since(start)

		if logger.discard(c) {
			return err
		}

		req := c.Request()
		res := c.Response()

		l := logger.alog.With(
			slog.String("id", logger.getRequestID(c)),
			slog.String("path", logger.sanitizePath(req.URL.Path)),
			slog.String("method", req.Method),
			slog.String("uri", req.RequestURI),
			slog.Int64("bytes_in", logger.parseContentLength(req.Header.Get(echo.HeaderContentLength))),
			slog.Int64("bytes_out", res.Size),
			slog.Any("remote_ip", strings.Split(c.RealIP(), ",")),
			slog.Int("status", res.Status),
			slog.String("host", req.Host),
			slog.String("referer", req.Referer()),
			slog.String("ua", req.UserAgent()),
			slog.String("route", c.Path()),
			slog.Duration("latency_ns", elapsed),
			slog.Any("tags", req.Header.Values(textproto.CanonicalMIMEHeaderKey("X-Tags"))),
		)

		if err != nil {
			l.Errorf("%s %s: %+v", req.Method, req.RequestURI, err)
			return
		}

		if res.Status >= 500 && res.Status < 600 {
			l.Errorf("%s %s", req.Method, req.RequestURI)
			return
		}

		l.Infof("%s %s", req.Method, req.RequestURI)

		return err
	}
}

type AccessLogLevel uint

const (
	// AccessLogLevelInfo will log every status code.
	AccessLogLevelVerbose = iota
	// AccessLogLevelInfo will not log 3XX and 404 status codes.
	AccessLogLevelInfo
	// AccessLogLevelWarn will log only 4XX (except 404) and 5XX status codes.
	AccessLogLevelWarn
	// AccessLogLevelError will log only 5XX status codes.
	AccessLogLevelError
)

type AccessLogDiscarder struct {
	Level        AccessLogLevel
	IgnoreRoutes *regexp.Regexp
}

func NewAccessLogDiscarder(level AccessLogLevel, routeFilter *regexp.Regexp) func(c Context) bool {
	return AccessLogDiscarder{
		Level:        level,
		IgnoreRoutes: routeFilter,
	}.Discard
}

func (discarder AccessLogDiscarder) Discard(c Context) bool {
	status := c.Response().Status
	switch discarder.Level {
	case AccessLogLevelError:
		if status < 500 {
			return true
		}
	case AccessLogLevelWarn:
		if status == http.StatusNotFound || status < 400 {
			return true
		}
	case AccessLogLevelInfo:
		if status == http.StatusNotFound || status >= 300 && status < 400 {
			return true
		}
	}
	if discarder.IgnoreRoutes != nil {
		discarder.IgnoreRoutes.MatchString(c.Request().URL.Path)
	}
	return false
}
