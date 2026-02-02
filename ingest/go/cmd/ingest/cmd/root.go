/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	telemetryPath string
	display       bool
)

var rootCmd = &cobra.Command{
	Use:   "go",
	Short: "IRacting telemetry ingest",
	Long: `The telemetry ingest allows us to take data from our racing sim and IRacing session and visualise that.
	In traditional motorsports that would give better insights to the race engineer who can build off the data to improve the driver and car.
	
	The ingest service uploads all the sessions that are stored on your local machine to the IRacing dashboard service. It can be run in the background or as a one off.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&display, "display", "d", true, "terminal display of the ingest process")
	rootCmd.Flags().StringVarP(&telemetryPath, "telemetryPath", "p", "", "path to IRacing telemetry folder")
}
