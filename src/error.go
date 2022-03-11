package eduvpn

import "fmt"

// Error structures defined here are used throughout the code

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
