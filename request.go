package webservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RequestOption func(req *StreamRequester)

type StreamRequester struct {
	cli     Client
	headers http.Header
	timeout time.Duration
}

func (req StreamRequester) validate() error {
	if req.cli.conn == nil {
		return fmt.Errorf("request must be created from a Client")
	}
	return nil
}

func (req StreamRequester) Clone() StreamRequester {
	return StreamRequester{
		cli:     req.cli,
		headers: req.headers.Clone(),
		timeout: req.timeout,
	}
}

func (req StreamRequester) WithHeader(key, value string) StreamRequester {
	req = req.Clone()
	req.headers.Add(key, value)
	return req
}

func (req StreamRequester) WithUniqueHeader(key, value string) StreamRequester {
	req = req.Clone()
	req.headers.Set(key, value)
	return req
}

func (req StreamRequester) WithHeaders(headers map[string]string) StreamRequester {
	req = req.Clone()
	for k, v := range headers {
		req.headers.Set(k, v)
	}
	return req
}

func (req StreamRequester) WithTimeout(timeout time.Duration) StreamRequester {
	req.timeout = timeout
	return req
}

// Prepare the request and return the underlying *http.Request to be used in other connections.
func (req StreamRequester) Prepare(ctx context.Context, method string, endpoint string, body io.Reader) (request *http.Request, err error) {
	if err = req.validate(); err != nil {
		return nil, err
	}

	hreq, err := http.NewRequestWithContext(ctx, method, req.cli.FullURL(endpoint), body)
	if err != nil {
		return nil, fmt.Errorf("error creating request; %w", err)
	}
	hreq.Header = req.headers

	for i := range req.cli.middlewares {
		if hreq, err = req.cli.middlewares[i](ctx, hreq); err != nil {
			return nil, fmt.Errorf("failed to run middleware [%d]; %w", i, err)
		}
	}

	return hreq, nil
}

// Do a stream request which will read the request body and will return the response as a ReadCloser.
// Callers must close the response.
// Request timeout includes reading the response is included in the timeout yet that is out of the scope of this method.
func (req StreamRequester) Do(ctx context.Context, method string, endpoint string, body io.Reader) (status int, response io.ReadCloser, err error) {
	hreq, err := req.Prepare(ctx, method, endpoint, body)
	if err != nil {
		return 0, nil, fmt.Errorf("error creating request; %w", err)
	}
	res, err := req.cli.conn.Do(hreq)
	if err != nil {
		return 0, nil, fmt.Errorf("error running request; %w", err)
	}
	return res.StatusCode, res.Body, nil
}

func (req StreamRequester) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.timeout > 0 {
		return context.WithTimeout(ctx, req.timeout)
	}
	return ctx, nil
}

// Requester ...
type Requester struct {
	core StreamRequester
}

func (req Requester) WithTimeout(timeout time.Duration) Requester {
	req.core = req.core.WithTimeout(timeout)
	return req
}

func (req Requester) WithHeader(key, value string) Requester {
	req.core = req.core.WithHeader(key, value)
	return req
}

func (req Requester) WithUniqueHeader(key, value string) Requester {
	req.core = req.core.WithUniqueHeader(key, value)
	return req
}

func (req Requester) WithHeaders(headers map[string]string) Requester {
	req.core = req.core.WithHeaders(headers)
	return req
}

// Prepare the request and return the underlying *http.Request to be used in other connections.
// Note that the provided context will not be wrapped by a new context with the configured request timeout.
func (req Requester) Prepare(ctx context.Context, method string, endpoint string, body []byte) (*http.Request, error) {
	return req.core.Prepare(ctx, method, endpoint, bytes.NewBuffer(body))
}

func (req Requester) Do(ctx context.Context, method string, endpoint string, data []byte) (status int, response []byte, err error) {
	ctx, cancel := req.core.Context(ctx)
	if cancel != nil {
		defer cancel()
	}

	status, payload, err := req.core.Do(ctx, method, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return 0, nil, err
	}
	defer payload.Close()

	response, err = io.ReadAll(payload)
	if err != nil {
		return status, nil, fmt.Errorf("error reading http body; %w", err)
	}

	return status, response, nil
}

type JSONRequester struct {
	core Requester
}

func (req JSONRequester) Prepare(ctx context.Context, method string, endpoint string, data interface{}) (*http.Request, error) {
	ebody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("invalid request body; %w", err)
	}
	return req.core.Prepare(ctx, method, endpoint, ebody)
}

func (req JSONRequester) Do(ctx context.Context, method string, endpoint string, data interface{}) (status int, response []byte, err error) {
	ebody, err := json.Marshal(data)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid request body; %w", err)
	}

	return req.core.Do(ctx, method, endpoint, ebody)
}

func (req JSONRequester) WithTimeout(timeout time.Duration) JSONRequester {
	req.core = req.core.WithTimeout(timeout)
	return req
}

func (req JSONRequester) WithHeader(key, value string) JSONRequester {
	req.core = req.core.WithHeader(key, value)
	return req
}

func (req JSONRequester) WithUniqueHeader(key, value string) JSONRequester {
	req.core = req.core.WithUniqueHeader(key, value)
	return req
}

func (req JSONRequester) WithHeaders(headers map[string]string) JSONRequester {
	req.core = req.core.WithHeaders(headers)
	return req
}
