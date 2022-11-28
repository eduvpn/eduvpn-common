// Package http defines higher level helpers for the net/http package
package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eduvpn/eduvpn-common/types"
)

// The URLParemeters as the name suggests is a type used for the parameters in the URL.
type URLParameters map[string]string

// OptionalParams is a structure that defines the optional parameters that are given when making a HTTP call.
type OptionalParams struct {
	Headers       http.Header
	URLParameters URLParameters
	Body          url.Values
	Timeout       time.Duration
}

// ConstructURL creates a URL with the included parameters.
func ConstructURL(baseURL string, parameters URLParameters) (string, error) {
	url, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		return "", types.NewWrappedError(
			fmt.Sprintf(
				"failed to construct url: %s including parameters: %v",
				url,
				parameters,
			),
			parseErr,
		)
	}

	q := url.Query()

	for parameter, value := range parameters {
		q.Set(parameter, value)
	}
	url.RawQuery = q.Encode()
	return url.String(), nil
}

// Get creates a Get request and returns the headers, body and an error.
func Get(url string) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodGet, url, nil)
}

// Post creates a Post request and returns the headers, body and an error.
func Post(url string, body url.Values) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodGet, url, &OptionalParams{Body: body})
}

// GetWithOpts creates a Get request with optional parameters and returns the headers, body and an error.
func GetWithOpts(url string, opts *OptionalParams) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodGet, url, opts)
}

// PostWithOpts creates a Post request with optional parameters and returns the headers, body and an error.
func PostWithOpts(url string, opts *OptionalParams) (http.Header, []byte, error) {
	return MethodWithOpts(http.MethodPost, url, opts)
}

// optionalURL ensures that the URL contains the optional parameters
// it returns the url (with parameters if success) and an error indicating success.
func optionalURL(url string, opts *OptionalParams) (string, error) {
	if opts != nil {
		url, urlErr := ConstructURL(url, opts.URLParameters)

		if urlErr != nil {
			return url, types.NewWrappedError(
				fmt.Sprintf("failed to create HTTP request with url: %s", url),
				urlErr,
			)
		}
		return url, nil
	}
	return url, nil
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

// MethodWithOpts creates a HTTP request using a method (e.g. GET, POST), an url and optional parameters
// It returns the HTTP headers, the body and an error if there is one.
func MethodWithOpts(
	method string,
	url string,
	opts *OptionalParams,
) (http.Header, []byte, error) {
	// Make sure the url contains all the parameters
	// This can return an error,
	// it already has the right error so we don't wrap it further
	url, urlErr := optionalURL(url, opts)
	if urlErr != nil {
		// No further type wrapping is needed here
		return nil, nil, urlErr
	}

	// Default timeout is 5 seconds
	// If a different timeout is given, set it
	var timeout time.Duration = 5
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	// Create a client
	client := &http.Client{Timeout: timeout * time.Second}

	errorMessage := fmt.Sprintf("failed HTTP request with method %s and url %s", method, url)

	// Create request object with the body reader generated from the optional arguments
	req, reqErr := http.NewRequest(method, url, optionalBodyReader(opts))
	if reqErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, reqErr)
	}

	// See https://stackoverflow.com/questions/17714494/golang-http-request-results-in-eof-errors-when-making-multiple-requests-successi
	req.Close = true

	// Make sure the headers contain all the parameters
	optionalHeaders(req, opts)

	// Do request
	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, respErr)
	}

	// Request successful, make sure body is closed at the end
	defer resp.Body.Close()

	// Return a string
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return resp.Header, nil, types.NewWrappedError(errorMessage, readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// We make this a custom error because we want to extract the status code later
		statusErr := &StatusError{URL: url, Body: string(body), Status: resp.StatusCode}
		return resp.Header, body, types.NewWrappedError(errorMessage, statusErr)
	}

	// Return the body in bytes and signal the status error if there was one
	return resp.Header, body, nil
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

// ParseJSONError indicates that the HTTP error is because of failed JSON parsing
// It has the URL and the Body as context.
// The underlying JSON parsing Err itself is also wrapped here.
type ParseJSONError struct {
	URL  string
	Body string
	Err  error
}

// Error returns the ParseJSONError as an error string.
func (e *ParseJSONError) Error() string {
	return fmt.Sprintf(
		"failed parsing json %s for HTTP resource: %s with error: %v",
		e.Body,
		e.URL,
		e.Err,
	)
}
