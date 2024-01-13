package resource

import (
	"time"
)

// options that can be applied to resource.Request
//
// option functions begin with "OptionRequest"
type requestOption interface {
	ApplyOption(*Request)
}

// sets the client for resource.Request
func OptionRequestClient(r Requester) requestOption {
	return optRequestClient{r}
}

// sets the request timeout for resource.Request
//
// argument taken in is a string but on ApplyOption() will be
// parsed to a time.Duration
//
// note: this value will only be used if the Request.client is
// of type *http.Client
func OptionRequestTimeout(d string) requestOption {
	return optRequestTimeout(d)
}

// sets the url to request in resource.Request
func OptionRequestUrl(url string) requestOption {
	return optRequestUrl(url)
}

// private option type to satisfy requestOption
type (
	optRequestClient struct {
		Requester
	}
	optRequestTimeout string
	optRequestUrl     string
)

func (c optRequestClient) ApplyOption(r *Request) {
	r.client = c.Requester
}

func (t optRequestTimeout) ApplyOption(r *Request) {
	parsedTimeout, err := time.ParseDuration(string(t))
	if err != nil {
		panic(err)
	}

	r.timeout = parsedTimeout
}

func (u optRequestUrl) ApplyOption(r *Request) {
	r.url = string(u)
}
