# rtime - a tool for request timing and analysis
[![Go](https://github.com/rdenson/rtime/actions/workflows/go-release.yml/badge.svg?branch=0.0.2-beta)](https://github.com/rdenson/rtime/actions/workflows/go-release.yml)

rtime (_requestTime_) attempts to request a specified URL to determine how fast
your pages or endpoints are loading.

## Two Paths - Request Scenarios
1. GET a resource that we know will return some HTML
   * see if we can resolve any known resources referenced in the returned HTML
   * estimate the timing of the total resources requested
2. GET an arbitrary resource

In addition to timing a request, you can see some general information about the
request:
* headers returned in the response
* TLS information
* additional resources requested (_see path 1 above_)

### Notes
This code is written to be a diagnostic tool and is still being developed.

### Command Completion
An example of how to take advantage of the command completion for this program.
rtime uses cobra, more examples specific to your shell can be found 👉  [here](https://github.com/spf13/cobra/blob/master/shell_completions.md).
```sh
# on mac os + zsh
# assuming download from releases
cd ~/Downloads/rtime_<tag>_os_arch/
cp rtime /usr/local/bin/
# set up command completion
# see https://github.com/spf13/cobra/blob/master/shell_completions.md (zsh section)
./rtime completion zsh > _rtime
cp _rtime ~/.zsh/functions

# path above used because in my .zshrc:
fpath=(~/.zsh/functions $fpath)
```
