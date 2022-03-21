package eduvpn

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type HTTPResourceError struct {
	URL string
	Err error
}

func (e *HTTPResourceError) Error() string {
	return fmt.Sprintf("failed obtaining HTTP resource %s with error %v", e.URL, e.Err)
}

type HTTPStatusError struct {
	URL    string
	Status int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("failed obtaining HTTP resource %s as it gave an unsuccesful status code %d", e.URL, e.Status)
}

type HTTPReadError struct {
	URL string
	Err error
}

func (e *HTTPReadError) Error() string {
	return fmt.Sprintf("failed reading HTTP resource %s with error %v", e.URL, e.Err)
}

type HTTPParseJsonError struct {
	URL  string
	Body string
	Err  error
}

func (e *HTTPParseJsonError) Error() string {
	return fmt.Sprintf("failed parsing json %s for HTTP resource %s with error %v", e.Body, e.URL, e.Err)
}

type HTTPRequestCreateError struct {
	URL string
	Err error
}

func (e *HTTPRequestCreateError) Error() string {
	return fmt.Sprintf("failed to create HTTP request with url %s and error %v", e.URL, e.Err)
}

type URLParameters map[string]string

type HTTPOptionalParams struct {
	Headers       *http.Header
	URLParameters *URLParameters
	Body          url.Values
}

// Construct an URL including on parameters
func HTTPConstructURL(baseURL string, parameters URLParameters) (string, error) {
	url, err := url.Parse(baseURL)

	if err != nil {
		return "", err
	}

	q := url.Query()

	for parameter, value := range parameters {
		q.Set(parameter, value)
	}
	url.RawQuery = q.Encode()
	return url.String(), nil
}

// Convenience functions
func HTTPGet(url string) ([]byte, error) {
	return HTTPMethodWithOpts(http.MethodGet, url, nil)
}

func HTTPPost(url string, body url.Values) ([]byte, error) {
	return HTTPMethodWithOpts(http.MethodGet, url, &HTTPOptionalParams{Body: body})
}

func HTTPGetWithOpts(url string, opts *HTTPOptionalParams) ([]byte, error) {
	return HTTPMethodWithOpts(http.MethodGet, url, opts)
}

func HTTPPostWithOpts(url string, opts *HTTPOptionalParams) ([]byte, error) {
	return HTTPMethodWithOpts(http.MethodPost, url, opts)
}

func httpOptionalURL(url string, opts *HTTPOptionalParams) (string, error) {
	if opts != nil && opts.URLParameters != nil {
		url, urlErr := HTTPConstructURL(url, *opts.URLParameters)

		if urlErr != nil {
			return url, &HTTPRequestCreateError{URL: url, Err: urlErr}
		}
		return url, nil
	}
	return url, nil
}

func httpOptionalHeaders(req *http.Request, opts *HTTPOptionalParams) {
	// Add headers
	if opts != nil && opts.Headers != nil && req != nil {
		for k, v := range *opts.Headers {
			req.Header.Add(k, v[0])
		}
	}

}

func httpOptionalBodyReader(opts *HTTPOptionalParams) io.Reader {
	if opts != nil && opts.Body != nil {
		return strings.NewReader(opts.Body.Encode())
	}
	return nil
}

func HTTPMethodWithOpts(method string, url string, opts *HTTPOptionalParams) ([]byte, error) {

	// Make sure the url contains all the parameters
	// This can return an error,
	// it already has the right error so so we don't wrap it further
	url, urlErr := httpOptionalURL(url, opts)
	if urlErr != nil {
		return nil, urlErr
	}

	// Create a client
	client := &http.Client{}

	// Create request object with the body reader generated from the optional arguments
	req, reqErr := http.NewRequest(method, url, httpOptionalBodyReader(opts))
	if reqErr != nil {
		return nil, &HTTPRequestCreateError{URL: url, Err: reqErr}
	}

	// Make sure the headers contain all the parameters
	httpOptionalHeaders(req, opts)

	// Do request
	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, &HTTPResourceError{URL: url, Err: respErr}
	}

	// Request successful, make sure body is closed at the end
	defer resp.Body.Close()

	// Return a string
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &HTTPReadError{URL: url, Err: readErr}
	}

	// Return the body in bytes and signal that there was no error
	return body, nil
}
