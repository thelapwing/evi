package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thelapwing/evi/internal/evidock"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a workspace with evidence.yaml and README",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := evidock.NewWorkspaceService()
		if err := svc.Init(workspaceDir); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"Workspace initialized: %s\nCreated or preserved: %s, %s\n",
			filepath.Clean(workspaceDir),
			filepath.Join(workspaceDir, "evidence.yaml"),
			filepath.Join(workspaceDir, "README.md"))
		return err
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
