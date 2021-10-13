package main

import (
	"context"
	"github.com/spf13/cobra"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/server"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "server [flags]",
		Short: "starts the O-Neko URL trigger server",
		Run: func(cmd *cobra.Command, args []string) {
			appContext, appContextCancel := context.WithCancel(context.Background())
			defer appContextCancel()
			triggerServer := server.New(config.Configuration(), appContext, GetVersion())
			triggerServer.Start()
		},
	})
}
