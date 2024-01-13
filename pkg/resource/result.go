package resource

import (
	"fmt"
	"net/http"
	"time"
)

type Result struct {
	Err          error
	RequestedUrl string
	Response     *http.Response
	Status       int
	Timing       time.Duration
}

func (rr *Result) PrettyPrint() {
	if rr.Err == nil {
		fmt.Printf(
			"%5s %s %6d - %s\n", " ",
			rr.Timing,
			rr.Status,
			rr.RequestedUrl,
		)
	} else {
		fmt.Printf("%+v\n", rr.Err)
	}
}

func (rr *Result) SetStatusFromResponse() {
	if rr.Err == nil {
		rr.Status = rr.Response.StatusCode
	}
}

func (rr *Result) SetTiming(d time.Duration) {
	rr.Timing = d
}
