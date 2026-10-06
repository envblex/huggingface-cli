package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root huggingface-cli Cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "huggingface-cli",
		Short: "Hugging Face Hub Command Line Interface",
		Long:  "huggingface-cli command helpers for interacting with the Hugging Face Hub, downloading models, datasets, managing cache, and authentication.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newEnvCmd())
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newWhoamiCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newRepoCmd())
	cmd.AddCommand(newUploadCmd())
	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newScanCacheCmd())
	cmd.AddCommand(newDeleteCacheCmd())
	cmd.AddCommand(newLFSEnableCmd())
	cmd.AddCommand(newLFSUploadCmd())

	return cmd
}
