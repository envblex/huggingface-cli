package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Print information about the environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			hf.DumpEnvironmentInfo(os.Stdout)
			return nil
		},
	}
}

func newLFSEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lfs-enable-largefiles <path>",
		Short: "Configure your repository to enable upload of files > 5GB.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return hf.LFSEnableLargefiles(args[0])
		},
	}
}

func newLFSUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "lfs-multipart-upload",
		Short:  "Internal command called by git-lfs for multipart transfers",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hf.LFSMultipartUpload(os.Stdin, os.Stdout)
		},
	}
	return cmd
}
