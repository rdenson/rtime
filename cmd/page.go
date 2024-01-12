package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/rdenson/rtime/pkg/resource"
	"github.com/spf13/cobra"
)

var pageCmd = &cobra.Command{
	Use:   "page [url]",
	Short: "request a page, just as you would in your browser",
	Long: `Attempts to request the specified URL and resolve any associated
  resources. eg. css, images, scripts, etc. Requests for additional
  resources are made concurrently.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		req, newRequestErr := resource.NewRequest(args[0])
		if newRequestErr != nil {
			return newRequestErr
		}

		req.SetRedirectsToPrint()
		// fmt.Printf("initially requesting: %s\n", req.Url)
		resp, reqResult := req.Exec()
		fmt.Printf("request took: %s\n", reqResult.Timing)
		if reqResult.RequestErr != nil {
			return reqResult.RequestErr
		}

		fmt.Printf("status: %s\n", resp.Status)

		resourcesToResolve := getResourcesFromResponseBody(resp.Body)
		fmt.Println("resolving resources...")
		timings := make([]*resource.Result, 0)
		timingCh := make(chan *resource.Result, 1)
		resourceWg := new(sync.WaitGroup)
		req.UnsetCheckRedirect()
		go func(wg *sync.WaitGroup) {
			for {
				reqResult, isOpen := <-timingCh
				if !isOpen {
					break
				}

				wg.Done()
				timings = append(timings, reqResult)
			}
		}(resourceWg)
		for _, resourcePath := range resourcesToResolve {
			resourceWg.Add(1)
			fmt.Printf(">>> requesting %s\n", resourcePath)
			go req.ExecAsync(
				timingCh,
				// resourceWg,
			)
		}

		resourceWg.Wait()
		fmt.Printf(
			"%4sfinished requesting %d resource(s)\n",
			" ",
			len(timings),
		)
		close(timingCh)
		var largestResourceRequestTime time.Duration
		for _, t := range timings {
			if t.Timing > largestResourceRequestTime {
				largestResourceRequestTime = t.Timing
			}
		}

		fmt.Printf("%4slongest associated resource request time: %s\n", " ", largestResourceRequestTime)
		fmt.Printf("total request estimated at %s\n", reqResult.Timing+largestResourceRequestTime)

		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(resp)
		}

		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(resp)
		}

		if showResourceRequests, _ := cmd.Flags().GetBool("show-resources-requested"); showResourceRequests {
			fmt.Println()
			fmt.Println("resources parsed from initial request body:")
			for _, t := range timings {
				fmt.Printf("%5s %s %5d - %s\n", " ", t.Timing, t.RequestStatus, t.ResourceUrl)
			}

			fmt.Println()
		}

		return nil
	},
}
