package main

import (
	"github.com/spf13/cobra"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/server"
)

var serverCmd *cobra.Command

func init() {
	serverCmd = &cobra.Command{
		Use: "server [flags]",
		Short: "starts the O-Neko URL trigger server",
		Run: func(cmd *cobra.Command, args []string) {
			server.Start(config.Configuration)
		},
	}
	rootCmd.AddCommand(serverCmd)
}
