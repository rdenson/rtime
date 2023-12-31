package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	ToolVersion string = "local"
	versionCmd         = &cobra.Command{
		Use:   "version",
		Short: "show the version",
		Long:  "...",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("⏱  rtime - request timing and analysis\n   version: %s\n", ToolVersion)
		},
	}
)
