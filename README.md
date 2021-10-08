# rtime - a tool for request timing and analysis
[![Go](https://github.com/rdenson/rtime/actions/workflows/go-release.yml/badge.svg?branch=0.0.2-beta)](https://github.com/rdenson/rtime/actions/workflows/go-release.yml)

rtime (_requestTime_) attempts to request a specified URL to determine how fast
your pages or endpoints are loading.

## Two Paths - Request Scenarios
1. GET a resource that we know will return some HTML
  * see if we can resolve any known resources referenced in the returned HTML
  * estimate the timing of the total resources requested
1. GET an arbitrary resource

In addition to timing a request, you can see some general information about the
request:
* headers returned in the response
* TLS information
* additional resources requested (_see path 1 above_)

### Notes
This tool is written to be a diagnostic and is still being developed.
