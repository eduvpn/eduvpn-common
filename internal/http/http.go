// Package http defines higher level helpers for the net/http package
package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-errors/errors"
)

// URLParameters is a type used for the parameters in the URL.
type URLParameters map[string]string

// OptionalParams is a structure that defines the optional parameters that are given when making a HTTP call.
type OptionalParams struct {
	Headers       http.Header
	URLParameters URLParameters
	Body          url.Values
	Timeout       time.Duration
}

// ConstructURL creates a URL with the included parameters.
func ConstructURL(baseURL string, params URLParameters) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.WrapPrefix(err,
			fmt.Sprintf("failed to construct url '%s' with parameters: %v", u, params), 0)
	}

	q := u.Query()

	for p, value := range params {
		q.Set(p, value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Get creates a Get request and returns the headers, body and an error.
func Get(url string) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodGet, url, nil)
}

// PostWithOpts creates a Post request with optional parameters and returns the headers, body and an error.
func PostWithOpts(url string, opts *OptionalParams) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodPost, url, opts)
}

// optionalURL ensures that the URL contains the optional parameters
// it returns the url (with parameters if success) and an error indicating success.
func optionalURL(urlStr string, opts *OptionalParams) (string, error) {
	if opts == nil {
		return urlStr, nil
	}

	return ConstructURL(urlStr, opts.URLParameters)
}

// optionalHeaders ensures that the HTTP request uses the optional headers if defined.
func optionalHeaders(req *http.Request, opts *OptionalParams) {
	// Add headers
	if opts != nil && req != nil && opts.Headers != nil {
		for k, v := range opts.Headers {
			req.Header.Add(k, v[0])
		}
	}
}

// optionalBodyReader returns a HTTP body reader if there is a body, otherwise nil.
func optionalBodyReader(opts *OptionalParams) io.Reader {
	if opts != nil && opts.Body != nil {
		return strings.NewReader(opts.Body.Encode())
	}
	return nil
}

// ReadLimit denotes the maximum amount of bytes that are read in HTTP responses
// This is used to prevent servers from sending huge amounts of data
// A limit of 16MB, although maybe much larger than needed, ensures that we do not run into problems
var ReadLimit int64 = 16 << 20

// MethodWithOpts creates a HTTP request using a method (e.g. GET, POST), an url and optional parameters
// It returns the HTTP headers, the body and an error if there is one.
func MethodWithOpts(
	method string,
	urlStr string,
	opts *OptionalParams,
) (http.Header, []byte, error) {
	// Make sure the url contains all the parameters
	// This can return an error,
	// it already has the right error, so we don't wrap it further
	urlStr, err := optionalURL(urlStr, opts)
	if err != nil {
		// No further type wrapping is needed here
		return nil, nil, err
	}

	// Default timeout is 5 seconds
	// If a different timeout is given, set it
	var timeout time.Duration = 5
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create request object with the body reader generated from the optional arguments
	req, err := http.NewRequest(method, urlStr, optionalBodyReader(opts))
	if err != nil {
		return nil, nil, errors.WrapPrefix(err,
			fmt.Sprintf("failed HTTP request with method %s and url %s", method, urlStr), 0)
	}

	// See https://stackoverflow.com/questions/17714494/golang-http-request-results-in-eof-errors-when-making-multiple-requests-successi
	req.Close = true

	// Make sure the headers contain all the parameters
	optionalHeaders(req, opts)

	// Create a client
	c := &http.Client{Timeout: timeout * time.Second}

	// Do request
	res, err := c.Do(req)
	if err != nil {
		return nil, nil, errors.WrapPrefix(err,
			fmt.Sprintf("failed HTTP request with method %s and url %s", method, urlStr), 0)
	}

	// Request successful, make sure body is closed at the end
	defer func() {
		_ = res.Body.Close()
	}()

	// Return a string
	// A max bytes reader is normally used for request bodies with a writer
	// However, this is still nice to use because unlike a limitreader, it returns an error if the body is too large
	// We use this function without a writer so we pass nil
	// We impose a limit because servers could be malicious and send huge amounts of data
	r := http.MaxBytesReader(nil, res.Body, ReadLimit)
	body, err := io.ReadAll(r)
	if err != nil {
		return res.Header, nil, errors.WrapPrefix(err,
			fmt.Sprintf("failed HTTP request with method: %s, url: %s and max bytes size: %v", method, urlStr, ReadLimit), 0)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return res.Header, body, errors.Wrap(&StatusError{URL: urlStr, Body: string(body), Status: res.StatusCode}, 0)
	}

	// Return the body in bytes and signal the status error if there was one
	return res.Header, body, nil
}

// StatusError indicates that we have received a HTTP status error.
type StatusError struct {
	URL    string
	Body   string
	Status int
}

// Error returns the StatusError as an error string.
func (e *StatusError) Error() string {
	return fmt.Sprintf(
		"failed obtaining HTTP resource: %s as it gave an unsuccessful status code: %d. Body: %s",
		e.URL,
		e.Status,
		e.Body,
	)
}
