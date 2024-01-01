package resource

import (
	"fmt"
	"time"
)

type Result struct {
	RequestErr    error
	RequestStatus int
	ResourceUrl   string
	Timing        time.Duration
}

func (rr *Result) PrettyPrint() {
	if rr.RequestErr == nil {
		fmt.Printf(
			"%5s %s %6d - %s\n", " ",
			rr.Timing,
			rr.RequestStatus,
			rr.ResourceUrl,
		)
	} else {
		fmt.Printf("%+v\n", rr.RequestErr)
	}
}

func (rr *Result) SetTiming(d time.Duration) {
	rr.Timing = d
}
