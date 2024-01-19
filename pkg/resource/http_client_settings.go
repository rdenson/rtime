package resource

import (
	"time"
)

const defaultTimeout time.Duration = 15 * time.Second

type httpClientSettings struct {
	timeout time.Duration
}

func newHttpClientSettings() *httpClientSettings {
	return &httpClientSettings{
		timeout: defaultTimeout,
	}
}
