package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	telemetryPath string
	display       bool
)

var rootCmd = &cobra.Command{
	Use:   "ingest",
	Short: "IRacting telemetry ingest",
	Long: `The telemetry ingest allows us to take data from our racing sim and IRacing session and visualise that.
	In traditional motorsports that would give better insights to the race engineer who can build off the data to improve the driver and car.
	
	The ingest service uploads all the sessions that are stored on your local machine to the IRacing dashboard service. It can be run in the background or as a one off.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd Run with args: %v \n", cmd)

		telemetryPath := cmd.Flag("telemetryPath").Value.String()

		if telemetryPath == "" {
			fmt.Println("no telemetry path found")
		} else {
			Process(telemetryPath)
		}

	},
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
	rootCmd.Flags().BoolVarP(&display, "display", "d", false, "terminal display of the ingest process")
	rootCmd.Flags().StringVarP(&telemetryPath, "telemetryPath", "p", "", "path to IRacing telemetry folder")
}
