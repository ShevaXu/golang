package web_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ShevaXu/golang/assert"
	"github.com/ShevaXu/golang/web"
)

func TestNewJSONPost(t *testing.T) {
	type testContent struct {
		Data string `json:"data"`
	}

	a := assert.NewAssert(t)

	req, err := web.NewJSONPost("/", testContent{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	a.Equal("application/json; charset=utf-8", req.Header.Get("Content-Type"), "Proper header")

	decoder := json.NewDecoder(req.Body)
	var c testContent
	err = decoder.Decode(&c)
	a.NoError(err, "Error decoding the body: %s")
	a.Equal(c.Data, "hello", `Should respond with "hello"`)
}

func DummyHandler(code int, content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ioutil.ReadAll(r.Body)
		w.WriteHeader(code)
		w.Write(content)
	}
}

func SleepHandler(d time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(d)
		w.WriteHeader(http.StatusOK)
	}
}

var (
	okResp    = []byte("OK")
	errResp   = []byte("exception")
	okHandler = DummyHandler(http.StatusOK, okResp)
)

type closeTest struct {
	h             http.Handler
	expectedCode  int
	expectedBody  []byte
	expectTimeout bool
}

// TODO: how to test if Close() works
func TestRequestWithClose(t *testing.T) {
	a := assert.NewAssert(t)

	tests := []closeTest{
		{
			okHandler,
			http.StatusOK,
			okResp,
			false,
		},
		{
			SleepHandler(20 * time.Millisecond),
			http.StatusOK,
			nil,
			true,
		},
	}

	cl := &http.Client{Timeout: 10 * time.Millisecond}

	for _, test := range tests {
		server := httptest.NewServer(test.h)

		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Errorf("Error new request: %s", err)
			continue
		}
		status, body, err := web.RequestWithClose(cl, req)
		if test.expectTimeout {
			a.NotNil(err, "Should return timeout error")
			a.Equal(true, web.IsTimeoutErr(err), "Should be a timeout")
		} else {
			a.NoError(err, "Request succeeds")
			a.Equal(test.expectedCode, status, "Returns code")
			a.Equal(test.expectedBody, body, "Returns body")
		}

		server.Close()
	}
}

func TestShouldRetry(t *testing.T) {
	a := assert.NewAssert(t)

	tests := []struct {
		code   int
		should bool
	}{
		{200, false},
		{400, false},
		{408, true},
		{500, true},
		{501, true},
		{502, true},
		{505, true},
		{511, true},
	}
	for _, test := range tests {
		a.Equal(test.should, web.ShouldRetry(test.code), "Retry test fails")
	}
}

func TestIsTimeoutErr(t *testing.T) {
	a := assert.NewAssert(t)

	tests := []struct {
		isTimeout bool
		err       error
		msg       string
	}{
		{false, errors.New("not timeout"), "Normal error is not"},
		{false, &net.AddrError{}, "AddrError is not"},
		{true, &net.DNSError{IsTimeout: true}, "DNSError is timeout"},
	}

	for _, test := range tests {
		a.Equal(test.isTimeout, web.IsTimeoutErr(test.err), test.msg)
	}
}

func TestBackoff_Next(t *testing.T) {
	a := assert.NewAssert(t)

	var (
		minTimeout  = 10
		maxTimeout  = 50
		testBackoff = web.Backoff{
			BaseSleep: minTimeout,
			MaxSleep:  maxTimeout,
		}
	)

	sleep0 := testBackoff.Next(0)
	a.True(sleep0 >= minTimeout && sleep0 <= minTimeout*3, "First sleep is bounded")
	sleep1 := testBackoff.Next(sleep0)
	sleep2 := testBackoff.Next(sleep1)
	sleep3 := testBackoff.Next(sleep2)
	a.True(sleep1 >= minTimeout && sleep2 >= minTimeout, "Each sleep > base")
	a.True(sleep2 <= maxTimeout && sleep3 <= maxTimeout, "Each sleep < max")
}

type retryTest struct {
	closeTest
	maxTries int
	tries    int
}

func TestClientDo(t *testing.T) {
	a := assert.NewAssert(t)

	tests := []retryTest{
		{
			closeTest{
				okHandler,
				http.StatusOK,
				okResp,
				false,
			},
			3,
			1,
		},
		{
			closeTest{
				DummyHandler(http.StatusInternalServerError, errResp),
				http.StatusInternalServerError,
				errResp,
				false,
			},
			5,
			5,
		},
	}

	cl := web.NewClient()

	for _, test := range tests {
		server := httptest.NewServer(test.h)

		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Errorf("Error new request: %s", err)
			continue
		}
		n, status, body, err := cl.Do(req, test.maxTries)
		if test.expectTimeout {
			a.NotNil(err, "Should return timeout error")
			a.Equal(true, web.IsTimeoutErr(err), "Should be a timeout")
		} else {
			a.NoError(err, "Request succeeds")
			a.Equal(test.expectedCode, status, "Returns code")
			a.Equal(test.expectedBody, body, "Returns body")
		}
		a.Equal(test.tries, n, "Report retried times")

		server.Close()
	}
}

type optionTest struct {
	cl web.Client
	retryTest
}

func TestClientOptions(t *testing.T) {
	a := assert.NewAssert(t)

	timeoutCl := web.NewClient(web.TimeoutOnly(),
		web.WithHTTPClient(&http.Client{Timeout: 10 * time.Millisecond}),
		web.WithBackoff(web.Backoff{BaseSleep: 100, MaxSleep: 200}))
	// TODO: case for backoff?
	tests := []optionTest{
		{
			timeoutCl,
			retryTest{
				closeTest{
					okHandler,
					http.StatusOK,
					okResp,
					false,
				},
				5,
				1,
			},
		},
		{
			timeoutCl,
			retryTest{
				closeTest{
					SleepHandler(50 * time.Millisecond),
					http.StatusOK,
					okResp,
					true,
				},
				5,
				5,
			},
		},
	}

	for _, test := range tests {
		server := httptest.NewServer(test.h)

		req, err := http.NewRequest("GET", server.URL, nil)
		if err != nil {
			t.Errorf("Error new request: %s", err)
			continue
		}
		n, status, body, err := test.cl.Do(req, test.maxTries)
		if test.expectTimeout {
			a.NotNil(err, "Should return timeout error")
			a.Equal(true, web.IsTimeoutErr(err), "Should be a timeout")
		} else {
			a.NoError(err, "Request succeeds")
			a.Equal(test.expectedCode, status, "Returns code")
			a.Equal(test.expectedBody, body, "Returns body")
		}
		a.Equal(test.tries, n, "Report retried times")

		server.Close()
	}
}

func TestClientDoPostWithBody(t *testing.T) {
	a := assert.NewAssert(t)

	tests := []retryTest{
		{
			closeTest{
				DummyHandler(http.StatusInternalServerError, errResp),
				http.StatusInternalServerError,
				errResp,
				false,
			},
			5,
			5,
		},
		{
			closeTest{
				SleepHandler(50 * time.Millisecond),
				http.StatusOK,
				okResp,
				true,
			},
			5,
			5,
		},
	}

	cl := web.NewClient(web.WithHTTPClient(&http.Client{Timeout: 10 * time.Millisecond}))

	for _, test := range tests {
		server := httptest.NewServer(test.h)

		req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer([]byte("foo-bar")))
		if err != nil {
			t.Errorf("Error new request: %s", err)
			continue
		}
		n, status, body, err := cl.Do(req, test.maxTries)
		if test.expectTimeout {
			a.NotNil(err, "Should return timeout error")
			a.Equal(true, web.IsTimeoutErr(err), fmt.Sprintf("Should be a timeout: %v", err))
		} else {
			a.NoError(err, "Request succeeds")
			a.Equal(test.expectedCode, status, "Returns code")
			a.Equal(test.expectedBody, body, "Returns body")
		}
		a.Equal(test.tries, n, "Report retried times")

		server.Close()
	}
}
