package webservice

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func NewMetricsMiddleware(register func(method, route, status string, elapsed time.Duration)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c Context) (err error) {
			start := time.Now()
			err = next(c)
			var req = c.Request()
			var res = c.Response()
			var method = req.Method
			var status = strconv.Itoa(res.Status)
			var route = c.Path()
			switch err {
			case echo.ErrNotFound:
				route = "ENOTFOUND"
			case echo.ErrMethodNotAllowed:
				route = "EMETHODNOTALLOWED"
			default:
				// placeholder because this should probably not store the route anyway
			}
			register(method, route, status, time.Since(start))

			return err
		}
	}
}
