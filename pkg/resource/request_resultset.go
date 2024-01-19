package resource

import (
	"net/http"
	"sync"
	"time"
)

// resultset key for request result subsets
const (
	setErrored    string = "erroredRequests"
	setSuccessful string = "successfulRequests"
)

// container for many request results
//
// This struct is intended to be used to analyze requests asynchronously.
// It executes a predefined request function (RequestMaker) and records
// the results.
type RequestResultset struct {
	ch        chan *RequestResult
	executor  RequestMaker
	resultset map[string][]*RequestResult
	wg        *sync.WaitGroup
}

// asynchronous requesting, results collected in an internal channel
//
// The intention is to ferry request targets into this function and
// then call RecieveResults() in a seperate goroutine to collect request
// results.
//
// note: this method call should be in the same goroutine as:
//   - NewRequestResultset()
//   - RequestResultset.Wait()
func (r *RequestResultset) Exec(client *http.Client, u string) {
	r.wg.Add(1)
	go func(ch chan *RequestResult, exec RequestMaker, hc *http.Client, url string) {
		ch <- exec(hc, url)
	}(r.ch, r.executor, client, u)
}

// get errored class for request results
func (r *RequestResultset) GetErroredResults() []*RequestResult {
	return r.resultset[setErrored]
}

// looks for the longest request among the successful request results
func (r *RequestResultset) GetLongestRequest() *RequestResult {
	var largestResourceRequestTime time.Duration
	var longestRequest *RequestResult

	for _, res := range r.resultset[setSuccessful] {
		if res.timing > largestResourceRequestTime {
			largestResourceRequestTime = res.timing
			longestRequest = res
		}
	}

	return longestRequest
}

// get successful class for request results
func (r *RequestResultset) GetSuccessfulResults() []*RequestResult {
	return r.resultset[setSuccessful]
}

// asynchronous request result collector
//
// Set this function call in a separate go routine. One the same goroutine
// as RequestResultset.Exec(), you'll need to call RequestResultset.Wait()
// to finish collecting results.
func (r *RequestResultset) ReceiveResults() {
	for {
		result, isOpen := <-r.ch
		if !isOpen {
			break
		}

		// fmt.Printf(">>> received: %s\n", result.Summarize())
		if result.err == nil {
			r.resultset[setSuccessful] = append(r.resultset[setSuccessful], result)
		} else {
			r.resultset[setErrored] = append(r.resultset[setErrored], result)
		}

		r.wg.Done()
	}
}

// waits for requests to finish
//
// Internally will cause ReceiveResults() to return.
func (r *RequestResultset) Wait() {
	r.wg.Wait()
	close(r.ch)
}

// initializes a new RequestResultset
//
// Input a predefined request function to execute requests. Caller
// needs to exec requests and gather results using the methods supplied
// from RequestResultset.
func NewRequestResultset(f RequestMaker) *RequestResultset {
	return &RequestResultset{
		ch:       make(chan *RequestResult, 1),
		executor: f,
		resultset: map[string][]*RequestResult{
			setErrored:    make([]*RequestResult, 0),
			setSuccessful: make([]*RequestResult, 0),
		},
		wg: new(sync.WaitGroup),
	}
}
