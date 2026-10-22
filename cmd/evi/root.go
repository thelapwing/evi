package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workspaceDir string

var rootCmd = &cobra.Command{
	Use:           "evi",
	Short:         "EviDock - manage evidence and verify source integrity",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workspaceDir, "workspace", "w", ".", "Path to the workspace directory")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
