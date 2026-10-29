package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thelapwing/evi/internal/evidock"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use: "validate", Short: "Validate evidence declarations and files",
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				count, err := evidock.NewWorkspaceService().Validate(workspaceDir)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Evidence valid: %d entries.\n", count)
				return err
			},
		},
		&cobra.Command{
			Use: "lock", Short: "Validate evidence and write a new SHA-256 lock",
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				lock, err := evidock.NewWorkspaceService().Lock(workspaceDir)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Lock saved: %d files; %s.\n", len(lock.Files), lock.LockedAt)
				return err
			},
		},
		&cobra.Command{
			Use: "verify", Short: "Compare evidence and sources with the lock without modifying it",
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				lock, err := evidock.NewWorkspaceService().Verify(workspaceDir)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Integrity verified: %d files.\n", len(lock.Files))
				return err
			},
		},
	)
}
