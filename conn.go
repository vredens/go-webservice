package webservice

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// DefaultConnOptions are the typicall connection options for the usual HTTP Client.
var DefaultConnOptions = ConnOptions{
	maxIdleConns:        5,
	maxIdleConnsPerHost: 5,
	maxConnsPerHost:     100,
	tcpKeepAlive:        30 * time.Second,
	keepAlive:           30 * time.Second,
	connTimeout:         10 * time.Second,
}

// ConnOptions data structure.
// Create an empty one and use the builder functions to add in your options.
type ConnOptions struct {
	maxIdleConns        int
	maxIdleConnsPerHost int
	maxConnsPerHost     int
	tcpKeepAlive        time.Duration
	keepAlive           time.Duration
	connTimeout         time.Duration
	requestTimeout      time.Duration
	dialerControl       *dialerControl
	tls                 *tls.Config
}

// WithMaxIdleConns sets the maximum idle connections left alive.
// Use this value to reduce the number of new connections created specially for bursts.
// Coordinate with a proper keep alive timeout for better results.
func (options ConnOptions) WithMaxIdleConns(value int) ConnOptions {
	options.maxIdleConns = value
	options.maxIdleConnsPerHost = value

	return options
}

// WithKeepAlive sets the amount of time an unused connection is kept open.
// This value should be as high as the period between bursts.
// Combine with the MaxIdleConns for better results.
func (options ConnOptions) WithKeepAlive(value time.Duration) ConnOptions {
	options.keepAlive = value
	options.tcpKeepAlive = value

	return options
}

// WithTimeout sets the timeout for new connections.
func (options ConnOptions) WithTimeout(value time.Duration) ConnOptions {
	options.connTimeout = value

	return options
}

// WithMaxConnsPerHost sets the maximum number of connections established to each host.
// This is only available from Go 1.11.
func (options ConnOptions) WithMaxConnsPerHost(value int) ConnOptions {
	options.maxConnsPerHost = value

	return options
}

// WithDialerHook allows providing a function which is called each time a dialer is executed.
func (options ConnOptions) WithDialerHook(host string, handler func(event DialerHookEvent)) ConnOptions {
	options.dialerControl = &dialerControl{hook: newDialerHook(host, handler)}

	return options
}

// WithRequestTimeout sets the maximum request timeout for all requests.
func (options ConnOptions) WithRequestTimeout(value time.Duration) ConnOptions {
	options.requestTimeout = value

	return options
}

func (options ConnOptions) WithTLSConfig(config *tls.Config) ConnOptions {
	options.tls = config
	return options
}

// DialerHookEvent data.
type DialerHookEvent struct {
	Msg     string
	Err     error
	Host    string
	Address string
	Lookups []net.IP
}

func newDialerHook(host string, handler func(event DialerHookEvent)) func(network, address string) {
	return func(network, address string) {
		u, err := url.Parse(host)
		if err != nil {
			handler(DialerHookEvent{Msg: "url parsing failed", Err: err, Host: host, Address: address})
			return
		}
		ips, err := net.LookupIP(u.Hostname())
		if err != nil {
			handler(DialerHookEvent{Msg: "dns lookup failed", Err: err, Host: host, Address: address})
			return
		}
		handler(DialerHookEvent{Msg: "dns lookup", Err: nil, Host: host, Address: address, Lookups: ips})
	}
}

func (options *ConnOptions) sanitize() {
	if options.keepAlive == 0 {
		options.keepAlive = 30 * time.Second
		options.tcpKeepAlive = 30 * time.Second
	}
	if options.maxIdleConns == 0 {
		options.maxIdleConns = 5
		options.maxIdleConnsPerHost = 5
	}
	if options.connTimeout == 0 {
		options.connTimeout = 3 * time.Second
	}
	if options.dialerControl == nil {
		options.dialerControl = defaultDialerController
	}
}

type dialerControl struct {
	hook func(network, address string)
}

func (dc *dialerControl) tap(network, address string, c syscall.RawConn) error {
	if dc.hook != nil {
		dc.hook(network, address)
	}
	return nil
}

var defaultDialerController = &dialerControl{}

// NewConn creates a new HTTP Connection with decent defaults or overriding them with the provided options.
func NewConn(opts ConnOptions) *http.Client {
	opts.sanitize()

	var dialer = &net.Dialer{
		KeepAlive: opts.tcpKeepAlive,
		Timeout:   opts.connTimeout, // default is 30s
		Control:   opts.dialerControl.tap,
	}
	var transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          opts.maxIdleConns,
		MaxIdleConnsPerHost:   opts.maxIdleConnsPerHost,
		IdleConnTimeout:       opts.keepAlive,
		TLSHandshakeTimeout:   opts.connTimeout + 100*time.Millisecond,
		ExpectContinueTimeout: opts.connTimeout + 100*time.Millisecond,
		MaxConnsPerHost:       opts.maxConnsPerHost,
	}
	if opts.tls != nil {
		transport.TLSClientConfig = opts.tls
	}

	return &http.Client{
		Transport: transport,
		Timeout:   opts.requestTimeout,
	}
}
