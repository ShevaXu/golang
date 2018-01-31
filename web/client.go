// Package web provides useful features to Go's http.Client
// following some best practices for production,
// such as timeout, retries and backoff.
package web

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// NewJSONPost returns a Request with json encoded and header set.
func NewJSONPost(url string, v interface{}) (*http.Request, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	return req, nil
}

// RequestWithClose sends the request and returns statusCode and raw body.
// It reads and closes Response.Body, return any error occurs.
func RequestWithClose(cl *http.Client, req *http.Request) (status int, body []byte, err error) {
	var resp *http.Response

	resp, err = cl.Do(req)
	// Close() iff resp did return
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return
	}

	status = resp.StatusCode

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}

// ShouldRetry determines if the client should repeat the request
// without modifications at any later time;
// returns true for http 408 and 5xx status.
func ShouldRetry(statusCode int) bool {
	// TODO: should exclude 501, 505 and 511?
	return statusCode == http.StatusRequestTimeout ||
		(statusCode >= 500 && statusCode <= 599)
}

// IsTimeoutErr checks if the error is a timeout.
func IsTimeoutErr(e error) bool {
	if err, ok := e.(net.Error); ok {
		return err.Timeout()
	}
	return false
}

// Backoff implements the exponential backoff algorithm
// with jitter for performing remote calls.
// It use an alternative method as described in
// https://www.awsarchitectureblog.com/2015/03/backoff.html.
type Backoff struct {
	BaseSleep, MaxSleep int
}

// Next returns the next sleep time computed by the previous one;
// the *Decorrelated Jitter* is:
// sleep = min(cap, random_between(base, sleep * 3)).
func (b *Backoff) Next(previous int) int {
	diff := previous*3 - b.BaseSleep
	// Intn will panic if arg <= 0
	if diff <= 0 {
		diff = 1
	}
	sleep := rand.Intn(diff) + b.BaseSleep
	if sleep > b.MaxSleep {
		return b.MaxSleep
	}
	return sleep
}

// Client provides additional features upon http.Client,
// e.g., io Reader handle and request retry with backoff.
type Client interface {
	// Do sends the request with at most maxTries time.
	// Retries happen in following conditions:
	// 1. timeout error occurs;
	// 2. should-retry status code is returned.
	// It also normalize the HTTP response as:
	// (tries, status int, body []byte, err error),
	// for #requests made, status code for the final request,
	// response body and error respectively.
	Do(req *http.Request, maxTries int) (tries, status int, body []byte, err error)
}

// client implements the Client interface.
// It wraps a http.Client underneath
// (safe for concurrent use by multiple goroutines).
type client struct {
	timeoutOnly bool // only retry for timeout error
	cl          *http.Client
	bk          Backoff
}

// NOTICE: retry works for request with no body only before go1.9.
func (c *client) Do(req *http.Request, maxTries int) (tries, status int, body []byte, err error) {
	// 0 will trigger setting wait to base
	wait := 0

	for tries = 1; tries <= maxTries; tries++ {
		// backoff
		time.Sleep(time.Duration(wait) * time.Millisecond)
		// update next sleep time
		wait = c.bk.Next(wait)
		// force reset Body if possible,
		// to avoid error: http: ContentLength=n with Body length 0
		if tries > 1 && req.Body != nil && req.GetBody != nil {
			req.Body, _ = req.GetBody()
		}
		// do request
		status, body, err = RequestWithClose(c.cl, req)
		if err != nil {
			if !c.timeoutOnly || IsTimeoutErr(err) {
				continue
			}
			return
		}
		// no error, check status
		if ShouldRetry(status) {
			continue
		}
		// succeed or should not repeat
		return
	}

	// return the last request's response, succeed or not
	tries--
	return
}

// ClientOption allows functional pattern options for client.
type ClientOption func(*client)

// TimeoutOnly sets the client to retry only
// on timeout error instead of all errors.
func TimeoutOnly() ClientOption {
	return func(c *client) {
		c.timeoutOnly = true
	}
}

// WithHTTPClient substitutes the default 5s
// timeout http.Client with a custom one.
func WithHTTPClient(cl *http.Client) ClientOption {
	return func(c *client) {
		c.cl = cl
	}
}

// WithBackoff substitutes the default Backoff.
func WithBackoff(b Backoff) ClientOption {
	return func(c *client) {
		c.bk = b
	}
}

// NewClient returns a client with default setting:
// 1. retry on all errors;
// 2. http.Client set Timeout to 5s;
// 3. Backoff{100, 5000}.
func NewClient(ops ...ClientOption) Client {
	c := &client{
		timeoutOnly: false, // retry all errors
		cl:          &http.Client{Timeout: 5 * time.Second},
		bk:          Backoff{100, 5000},
	}

	for _, op := range ops {
		op(c)
	}

	return c
}
