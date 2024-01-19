package resource

import (
	"time"
)

// option handler for setting up http clients
type httpClientOption interface {
	ApplyOption(*httpClientSettings)
}

type (
	optRequestTimeout string
)

// sets the request timeout for an http client
//
// Note that the argument taken in is a string but on ApplyOption()
// will be parsed to a time.Duration.
func OptionRequestTimeout(duration string) httpClientOption {
	return optRequestTimeout(duration)
}

func (t optRequestTimeout) ApplyOption(s *httpClientSettings) {
	parsedTimeout, err := time.ParseDuration(string(t))
	if err != nil {
		panic(err)
	}

	s.timeout = parsedTimeout
}
