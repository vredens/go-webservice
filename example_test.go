package webservice_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vredens/go-webservice"
)

func ExampleServer() {
	var srv = webservice.NewServer(":8001", webservice.ServerOptions{})
	srv.RegisterHealthRoutes("/_")
	srv.RegisterDebugRoutes("/_")
	srv.Echo.GET("/my/:path", func(ctx webservice.Context) error {
		return ctx.NoContent(200)
	})

	go srv.Start() // you should check the error here and add some form of routine management

	// something preventing the next lines of code to run right after

	if err := srv.Stop(); err != nil {
		panic(err)
	}
}

func ExampleNewClient() {
	// CREATING THE CLIENT
	var cli *webservice.Client

	// simple version, should be decent enough for everybody
	cli = webservice.NewClient("http://example.localhost:5000")

	// advanced version, a new client with the underlying http connection created above
	cli = webservice.NewCustomClient("http://example.localhost:5000", webservice.ClientOptions{
		Conn:              webservice.NewConn(webservice.DefaultConnOptions.WithTimeout(5 * time.Second)),
		MaxRequestTimeout: 30 * time.Second,
	})

	// add a dialer hook to get feedback each time a new tcp connection is created
	cli = webservice.NewCustomClient("http://example.localhost:5000", webservice.ClientOptions{
		Conn: webservice.NewConn(
			webservice.DefaultConnOptions.WithDialerHook("http://example.localhost:5000", func(event webservice.DialerHookEvent) {
				fmt.Printf("dialer event: %+v", event)
			}),
		),
		MaxRequestTimeout: 30 * time.Second, // set this to a high enough value
	})

	// DEFAULT HEADERS ON ALL REQUESTS

	cli.AddDefaultHeader("x-auth-token", "god")

	// REQUEST/RESPONSE

	var status int
	var response []byte
	var err error

	// simple version
	status, response, err = cli.NewRequest().Do(context.Background(), "POST", "/api/1/test", nil)

	// advanced version, using request builder pattern (each build call creates a clone, it is safe to reuse built requests)
	status, response, err = cli.NewRequest(
		cli.RequestTimeout(1*time.Second), // override the default client timeout
		cli.RequestHeaders(map[string]string{"key": "val"}),
	).Do(context.Background(), "POST", "/api/1/test", nil)

	// advanced version, using request options
	status, response, err = cli.NewRequest(
		cli.RequestTimeout(1*time.Second), // override the default client timeout
		cli.RequestHeaders(map[string]string{"key": "val"}),
	).Do(context.Background(), "POST", "/api/1/test", nil)

	// THE RESPONSE DATA

	if err != nil {
		fmt.Printf("error: %s", err)
	}
	fmt.Printf("status: %d", status)
	fmt.Printf("response: %s", string(response))
}

func ExampleNewConn() {
	// for creating an actual http.Client from go stdlib with some (hopefully) optimized parameters.
	var conn = webservice.NewConn(
		webservice.DefaultConnOptions.WithTimeout(1 * time.Second),
	)

	req, err := http.NewRequest(http.MethodGet, "http://example", nil)
	if err != nil {
		log.Fatalf("error creating req: %s", err)
	}
	res, err := conn.Do(req)
	if err != nil {
		log.Fatalf("error sending req: %s", err)
	}
	if res.StatusCode != http.StatusOK {
		log.Fatalf("poo")
	}
}
