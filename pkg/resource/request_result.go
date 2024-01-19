package resource

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const summaryFormat string = "%s%s%d | %dms"
const formatContentType string = " | %s | "
const formatNoContentType string = " | "

// response wrapper
//
// Slight specialization for an *http.Response. Adds timing
// information and combines error data when making the request.
type RequestResult struct {
	err        error
	initialUrl string
	response   *http.Response
	timing     time.Duration
}

func (r *RequestResult) GetError() error {
	return r.err
}

func (r *RequestResult) GetHeaders() http.Header {
	if r.response != nil {
		return r.response.Header
	}

	return nil
}

// looks for resources to resolve in a response body
//
// If you've requested a resource that returnes text/html, there
// may be included resources. This function returns them as a listing.
func (r *RequestResult) GetResourcesInResponse() []string {
	defer r.response.Body.Close()

	pageTokens := html.NewTokenizer(r.response.Body)
	finishedExaminingTokens := false
	bodyParsedResources := make([]string, 0)
	for !finishedExaminingTokens {
		if pageTokens.Next() == html.ErrorToken {
			finishedExaminingTokens = true
		}

		currentToken := pageTokens.Token()
		for _, a := range currentToken.Attr {
			if a.Key == "href" || a.Key == "src" {
				bodyParsedResources = append(bodyParsedResources, a.Val)
			}
		}
	}

	return bodyParsedResources
}

func (r *RequestResult) GetResponseUrl() string {
	if r.response != nil {
		return r.response.Request.URL.String()
	}

	return ""
}

func (r *RequestResult) GetTlsState() *tls.ConnectionState {
	return r.response.TLS
}

// gets the status code from the response
func (r *RequestResult) GetStatus() int {
	if r.response != nil {
		return r.response.StatusCode
	}

	return -1
}

func (r *RequestResult) GetTiming() time.Duration {
	if r != nil {
		return r.timing
	}

	return -1 * time.Millisecond
}

func (r *RequestResult) HasError() bool {
	if r != nil {
		return r.err != nil
	}

	return false
}

// pretty printed summation for the response
func (r *RequestResult) Summarize() string {
	if r == nil {
		return "unavailable"
	}

	if r.response == nil && r.HasError() {
		return fmt.Sprintf("error requesting [ %s ]: %s", r.initialUrl, r.err.Error())
	} else if r.response == nil {
		return fmt.Sprintf("no response from [ %s ]", r.initialUrl)
	}

	contentType := formatNoContentType
	contentTypeHeader := r.response.Header.Get("Content-Type")
	if len(contentTypeHeader) > 0 {
		contentType = fmt.Sprintf(
			formatContentType,
			strings.Split(contentTypeHeader, "; ")[0],
		)
	}

	return fmt.Sprintf(
		summaryFormat,
		r.initialUrl,
		contentType,
		r.response.StatusCode,
		r.timing.Milliseconds(),
	)
}
