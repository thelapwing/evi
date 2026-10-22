package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thelapwing/evi/internal/evidock"
)

var tidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "Restore missing evidence.yaml and README without overwriting files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := evidock.NewWorkspaceService()
		if err := svc.Tidy(workspaceDir); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Workspace restored: %s\n", filepath.Clean(workspaceDir))
		return err
	},
}

func init() {
	rootCmd.AddCommand(tidyCmd)
}
