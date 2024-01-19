package main

import (
	"fmt"

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
		showPageResources, _ := cmd.Flags().GetBool("show-resources-requested")
		hc := resource.GetHttpClient()
		res := resource.MakeRequest(hc, args[0])
		if res.HasError() {
			return res.GetError()
		}

		fmt.Printf(
			"%s responded in with %d in %.3fs\n",
			res.GetResponseUrl(),
			res.GetStatus(),
			res.GetTiming().Seconds(),
		)
		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(res.GetHeaders())
		}

		fmt.Print("trying to resolve page resources... ")
		resourcesToResolve := res.GetResourcesInResponse()
		fmt.Printf("resources found: %d\n", len(resourcesToResolve))

		asyncResults := resource.NewRequestResultset(resource.MakeRequest)
		go asyncResults.ReceiveResults()
		for _, resourcePath := range resourcesToResolve {
			asyncResults.Exec(hc, resourcePath)
		}

		asyncResults.Wait()

		if showPageResources {
			errs := asyncResults.GetErroredResults()
			rok := asyncResults.GetSuccessfulResults()
			fmt.Println("resource requests")
			fmt.Printf("%*serrors (%d)\n", 2, " ", len(errs))
			for _, e := range errs {
				fmt.Printf("%*s%s\n", 2, " ", e.Summarize())
			}

			fmt.Printf("%*sresponses (%d)\n", 2, " ", len(rok))
			for _, r := range rok {
				fmt.Printf("%*s%s\n", 2, " ", r.Summarize())
			}
		}

		longestRequest := asyncResults.GetLongestRequest()
		fmt.Printf("longest request: %s\n", longestRequest.Summarize())
		fmt.Printf("total request estimated at %s\n", res.GetTiming()+longestRequest.GetTiming())
		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(res.GetTlsState())
		}

		return nil
	},
}
