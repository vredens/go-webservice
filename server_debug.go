package webservice

import (
	"expvar"
	"log"
	"net/http"
	netpprof "net/http/pprof"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
)

// RegisterDebugRoutes registers preset handlers for <prefix>/debug/profile/cpu and <prefix>/debug/profile/mem
func (srv *Server) RegisterDebugRoutes(prefix string, middlewares ...echo.MiddlewareFunc) {
	group := srv.Echo.Group(prefix+"/debug/", middlewares...)
	group.GET("/profile", srv.handleRedirectToIndex)
	group.GET("/profile/", srv.handleProfileIndex)
	group.GET("/profile/cpu", srv.handleCPUProfiler)
	group.GET("/profile/cmdline", srv.handleCmdLineProfiler)
	group.GET("/profile/symbol", srv.handleSymbolProfiler)
	group.GET("/profile/trace", srv.handleTraceProfiler)
	group.GET("/profile/:profile", srv.handleProfile)
	group.GET("/expvar", srv.handleExpVar)
}

var profileDescriptions = map[string]string{
	"allocs":       "A sampling of all past memory allocations.",
	"block":        "Stack traces that led to blocking on synchronization primitives during a 5 second sampling period. Query params: rate=100 (sampling rate is 1/rate).",
	"cmdline":      "The command line invocation of the current program.",
	"goroutine":    "Stack traces of all current goroutines.",
	"heap":         "A sampling of memory allocations of live objects. Query params: gc=0 (if 1 calls GC before).",
	"mutex":        "Stack traces of holders of contended mutexes during a 5 second sampling period. Query params: rate=100 (sampling rate is 1/rate).",
	"cpu":          "CPU profile. After you get the profile file, use the go tool pprof command to investigate the profile. Query params: seconds=30 (sampling time)",
	"threadcreate": "Stack traces that led to the creation of new OS threads",
	"trace":        "A trace of execution of the current program. After you get the trace file, use the go tool trace command to investigate the trace. Query params: seconds=30 (sampling time)",
}

func (srv *Server) handleRedirectToIndex(c Context) error {
	return c.Redirect(http.StatusMovedPermanently, c.Request().RequestURI+"/")
}

func (srv *Server) handleExpVar(c Context) error {
	expvar.Handler().ServeHTTP(c.Response(), c.Request())

	return nil
}

func (srv *Server) handleProfileIndex(c Context) error {
	type profile struct {
		Name  string
		Href  string
		Desc  string
		Count int
	}
	var profiles []profile
	for _, p := range pprof.Profiles() {
		profiles = append(profiles, profile{
			Name:  p.Name(),
			Href:  p.Name() + "?debug=1",
			Desc:  profileDescriptions[p.Name()],
			Count: p.Count(),
		})
	}

	// Adding other profiles exposed from within this package
	for _, p := range []string{"cmdline", "cpu", "trace"} {
		profiles = append(profiles, profile{
			Name: p,
			Href: p,
			Desc: profileDescriptions[p],
		})
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	if err := indexTmpl.Execute(c.Response(), profiles); err != nil {
		log.Print(err)
	}
	return nil
}

func (srv *Server) handleCPUProfiler(c Context) error {
	netpprof.Profile(c.Response(), c.Request())

	return nil
}

func (srv *Server) handleTraceProfiler(c Context) error {
	netpprof.Trace(c.Response(), c.Request())

	return nil
}

func (srv *Server) handleSymbolProfiler(c Context) error {
	netpprof.Symbol(c.Response(), c.Request())

	return nil
}

func (srv *Server) handleCmdLineProfiler(c Context) error {
	netpprof.Cmdline(c.Response(), c.Request())

	return nil
}

func getIntQueryParam(c Context, name string, def, min, max int) int {
	val, _ := strconv.Atoi(c.QueryParam(name))
	if val == 0 {
		return def
	}
	if val < min || val > max {
		return def
	}
	return val
}

func (srv *Server) handleProfile(c Context) error {
	name := c.Param("profile")

	switch name {
	case "block":
		runtime.SetBlockProfileRate(getIntQueryParam(c, "rate", 100, 1, 100))
		defer runtime.SetBlockProfileRate(0)
		time.Sleep(5 * time.Second)
	case "mutex":
		runtime.SetMutexProfileFraction(getIntQueryParam(c, "rate", 100, 1, 100))
		defer runtime.SetMutexProfileFraction(0)
		time.Sleep(5 * time.Second)
	}

	netpprof.Handler(name).ServeHTTP(c.Response(), c.Request())

	return nil
}

var indexTmpl = template.Must(template.New("index").Parse(`

<head>
	<title>/debug/profile/</title>
	<style>
		.profile-name{
			display:inline-block;
			width:6rem;
		}
	</style>
</head>
<body>
	/debug/pprof/<br>
	<br>
	Types of profiles available:
	<table>
		<thead>
			<td>Count</td>
			<td>Profile</td>
			<td>Desc</td>
		</thead>
		{{range .}}
			<tr>
				<td>{{.Count}}</td><td><a href={{.Href}}>{{.Name}}</a></td><td>{{.Desc}}</td>
			</tr>
		{{end}}
		<tr>
			<td>-</td><td><a href="goroutine?debug=2">goroutine (all)</a></td><td>Full goroutine stack dump (using debug=2). ACHTUNG: this can be huge.</td>
		</tr>
	</table>
</body>
</html>
`))
