package main

import (
	"fmt"

	"github.com/rdenson/rtime/pkg/resource"
	"github.com/rs/zerolog/log"
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

		pageLog := log.With().Str("request level", "initial").Logger()
		pageLog.Info().Msgf(
			"%s responded in with %d in %.3fs",
			res.GetResponseUrl(),
			res.GetStatus(),
			res.GetTiming().Seconds(),
		)
		if showHeaders, _ := cmd.Flags().GetBool("show-headers"); showHeaders {
			showResponseHeaders(res.GetHeaders())
		}

		resourcesToResolve := res.GetResourcesInResponse()
		pageLog.Info().Int("included resources", len(resourcesToResolve)).Msg("resolving requests...")
		asyncResults := resource.NewRequestResultset(resource.MakeRequest)
		go asyncResults.ReceiveResults()
		for _, resourcePath := range resourcesToResolve {
			asyncResults.Exec(hc, resourcePath)
		}

		asyncResults.Wait()

		if showPageResources {
			subResourceLog := log.With().Str("request level", "included").Logger()
			errs := asyncResults.GetErroredResults()
			rok := asyncResults.GetSuccessfulResults()
			subResourceLog.Info().Int("request errors", len(errs)).Send()
			for _, e := range errs {
				fmt.Printf("%*s%s\n", 2, " ", e.Summarize())
			}

			subResourceLog.Info().Int("request responses", len(rok)).Send()
			for _, r := range rok {
				fmt.Printf("%*s%s\n", 2, " ", r.Summarize())
			}
		}

		longestRequest := asyncResults.GetLongestRequest()
		log.Info().Str(
			"longest request", longestRequest.GetResponseUrl(),
		).Float64(
			"time (ms)", float64(longestRequest.GetTiming().Milliseconds()),
		).Msgf("total request estimated at %s", res.GetTiming()+longestRequest.GetTiming())
		if analyzeTls, _ := cmd.Flags().GetBool("analyze-tls"); analyzeTls {
			showResponseTlsInfo(res.GetTlsState())
		}

		return nil
	},
}
