package main

import (
	"context"
	"github.com/spf13/cobra"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/server"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "server [flags]",
		Short: "starts the O-Neko catnip server",
		Run: func(cmd *cobra.Command, args []string) {
			appContext, appContextCancel := context.WithCancel(context.Background())
			defer appContextCancel()
			triggerServer := server.New(config.Configuration(), appContext, GetVersion())
			triggerServer.Start()
		},
	})
}
