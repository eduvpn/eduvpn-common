package eduvpn

import (
	"fmt"
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

type HTTPOptionalParams struct {
	Headers *http.Header
}

func HTTPGet(url string) ([]byte, error) {
	return HTTPGetWithOptionalParams(url, nil)
}

func HTTPGetWithOptionalParams(url string, opts *HTTPOptionalParams) ([]byte, error) {
	client := &http.Client{}
	req, reqErr := http.NewRequest(http.MethodGet, url, nil)
	if reqErr != nil {
		return nil, &HTTPRequestCreateError{URL: url, Err: reqErr}
	}
	if opts != nil && opts.Headers != nil {
		for k, v := range *opts.Headers {
			req.Header.Add(k, v[0])
		}
	}
	resp, respErr := client.Do(req)

	if respErr != nil {
		return nil, &HTTPResourceError{URL: url, Err: respErr}
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &HTTPReadError{URL: url, Err: readErr}
	}

	return body, nil
}

func HTTPPost(url string, body url.Values) ([]byte, error) {
	return HTTPPostWithOptionalParams(url, body, nil)
}

func HTTPPostWithOptionalParams(url string, data url.Values, opts *HTTPOptionalParams) ([]byte, error) {
	client := &http.Client{}
	req, reqErr := http.NewRequest(http.MethodPost, url, strings.NewReader(data.Encode()))
	if reqErr != nil {
		return nil, &HTTPRequestCreateError{URL: url, Err: reqErr}
	}
	if opts != nil && opts.Headers != nil {
		for k, v := range *opts.Headers {
			req.Header.Add(k, v[0])
		}
	}
	resp, respErr := client.Do(req)

	if respErr != nil {
		return nil, &HTTPResourceError{URL: url, Err: respErr}
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &HTTPReadError{URL: url, Err: readErr}
	}

	return body, nil
}
