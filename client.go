package webservice

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RequestMiddleware func(ctx context.Context, req *http.Request) (*http.Request, error)

// ClientOptions is the set of options for instancing a new Requester.
type ClientOptions struct {
	// Conn is the underlying http.Client to use.
	// You can use the NewConn method to assist in constructing this.
	Conn *http.Client
	// MaxRequestTimeout should be set high enough for this client.
	// Timeouts per request must be smaller than this value.
	// Use 0 to deactivate this timeout.
	MaxRequestTimeout time.Duration
	Headers           http.Header
	Middlewares       []RequestMiddleware
}

func (options ClientOptions) AddHeaders(headers map[string]string) ClientOptions {
	if options.Headers == nil {
		options.Headers = make(http.Header)
	}
	for k, v := range headers {
		options.Headers.Add(k, v)
	}
	return options
}

func (options ClientOptions) sanitize() ClientOptions {
	if options.MaxRequestTimeout == 0*time.Second {
		options.MaxRequestTimeout = 5 * time.Second
	}
	if options.Conn == nil {
		options.Conn = NewConn(DefaultConnOptions.WithRequestTimeout(options.MaxRequestTimeout).WithTimeout(options.MaxRequestTimeout))
	}
	if options.Headers == nil {
		options.Headers = make(http.Header)
	}
	return options
}

// Client is an HTTP Client wrapper for Go's official net/http client.
// Automatically constructs the UserAgent based on infrastructure information as best as possible.
type Client struct {
	host           string
	conn           *http.Client
	defaultTimeout time.Duration
	dheaders       http.Header
	middlewares    []RequestMiddleware
}

// NewClient creates a new Requester for a specific host
func NewClient(host string) *Client {
	return NewCustomClient(host, ClientOptions{})
}

// NewCustomClient creates a new Requester for a specific host with a supplied http transport
// Use this with caution ... should only be used to pass in custom Clients and Transports for testing purposes
func NewCustomClient(host string, options ClientOptions) *Client {
	options = options.sanitize()

	client := Client{
		host:           host,
		conn:           options.Conn,
		defaultTimeout: options.MaxRequestTimeout,
		dheaders:       options.Headers,
		middlewares:    options.Middlewares,
	}
	client.dheaders.Add("User-Agent", userAgent())

	if client.defaultTimeout > 0 && client.conn.Timeout != client.defaultTimeout {
		client.conn = &http.Client{
			CheckRedirect: options.Conn.CheckRedirect,
			Jar:           options.Conn.Jar,
			Transport:     options.Conn.Transport,
			Timeout:       client.defaultTimeout,
		}
	}

	return &client
}

// ClientStatusReport for clients.
type ClientStatusReport struct {
	Addresses    []net.IP      `json:"addresses,omitempty"`
	Error        string        `json:"error,omitempty"`
	PingDuration time.Duration `json:"elapsed_ns"`
}

// Ping returns a status report for the client's connection.
func (cli Client) Ping() ClientStatusReport {
	var start = time.Now()
	var _, err = cli.conn.Get(cli.host)
	var rep = ClientStatusReport{PingDuration: time.Since(start)}
	if err != nil {
		rep.Error = fmt.Sprintf("%+v", err)
	}
	u, err := url.Parse(cli.host)
	if err != nil {
		rep.Error = fmt.Sprintf("%+v", err)
		return rep
	}
	rep.Addresses, err = net.LookupIP(u.Hostname())
	if err != nil {
		rep.Error = fmt.Sprintf("%+v", err)
		return rep
	}
	return rep
}

func (cli Client) Clone() Client {
	return Client{
		host:           cli.host,
		conn:           cli.conn,
		defaultTimeout: cli.defaultTimeout,
		dheaders:       cli.dheaders.Clone(),
		middlewares:    cli.middlewares,
	}
}

func (cli Client) FullURL(endpoint string) string {
	return combineURL(cli.host, endpoint)
}

// AddDefaultHeader to all requests.
// If you wish to ensure only a single header value exists then use SetDefaultHeader instead.
func (cli *Client) AddDefaultHeader(key, value string) *Client {
	cli.dheaders.Add(key, value)
	return cli
}

// SetDefaultHeader to all requests will replace any default header with the same key.
func (cli *Client) SetDefaultHeader(key, value string) *Client {
	cli.dheaders.Set(key, value)
	return cli
}

// SetDefaultHeader to all requests will replace any default header with the same key.
func (cli *Client) SetTimeout(timeout time.Duration) *Client {
	cli.defaultTimeout = timeout
	if cli.defaultTimeout > 0 && cli.conn.Timeout != cli.defaultTimeout {
		cli.conn = &http.Client{
			CheckRedirect: cli.conn.CheckRedirect,
			Jar:           cli.conn.Jar,
			Transport:     cli.conn.Transport,
			Timeout:       cli.defaultTimeout,
		}
	}
	return cli
}

func (cli Client) RequestTimeout(timeout time.Duration) RequestOption {
	return func(req *StreamRequester) {
		req.timeout = timeout
	}
}

// RequestHeaders request option that sets unique headers when passed to a new request method.
// This can override any default headers in the client which are overwritten when passing onto the underlying request.
func (cli Client) RequestHeaders(headers map[string]string) RequestOption {
	return func(req *StreamRequester) {
		for k, v := range headers {
			req.headers.Set(k, v)
		}
	}
}

func (cli Client) RequestHeader(k, v string) RequestOption {
	return func(req *StreamRequester) {
		req.headers.Add(k, v)
	}
}

func (cli Client) RequestUniqueHeader(k, v string) RequestOption {
	return func(req *StreamRequester) {
		req.headers.Set(k, v)
	}
}

func (cli Client) WithDefaultHeader(k, v string) RequestOption {
	return func(req *StreamRequester) {
		if req.headers.Get(k) != "" {
			return
		}
		req.headers.Set(k, v)
	}
}

func (cli Client) NewStreamRequest(opts ...RequestOption) StreamRequester {
	var req = StreamRequester{
		cli:     cli,
		headers: cli.dheaders.Clone(),
	}
	for i := range opts {
		opts[i](&req)
	}
	return req
}

func (cli *Client) NewRequest(opts ...RequestOption) Requester {
	return Requester{
		core: cli.NewStreamRequest(opts...),
	}
}

func (cli *Client) NewJSONRequest(opts ...RequestOption) JSONRequester {
	var req = JSONRequester{
		core: cli.NewRequest(opts...),
	}
	req.core.core.headers.Set("Content-Type", "application/json")
	return req
}

// JSONRequest creates a http request with the body marshalled to JSON.
func (cli *Client) JSONRequest(ctx context.Context, method string, endpoint string, body interface{}) (int, []byte, error) {
	return cli.NewJSONRequest().Do(ctx, method, endpoint, body)
}

// Request creates a new http.Request and submits.
func (cli *Client) Request(ctx context.Context, method string, endpoint string, reqBody []byte) (status int, resBody []byte, err error) {
	return cli.NewRequest().Do(ctx, method, endpoint, reqBody)
}
