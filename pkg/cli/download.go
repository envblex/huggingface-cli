package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newDownloadCmd() *cobra.Command {
	var repoType string
	var revision string
	var include []string
	var exclude []string
	var cacheDir string
	var localDir string
	var localDirUseSymlinks string
	var forceDownload bool
	var resumeDownload bool
	var token string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "download <repo_id> [<filenames>...]",
		Short: "Download files from the Hub",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoID := args[0]
			filenames := args[1:]

			client := hf.NewClient("")

			if len(filenames) == 1 {
				outPath, err := client.DownloadFile(hf.DownloadFileOptions{
					RepoID:              repoID,
					RepoType:            repoType,
					Revision:            revision,
					Filename:            filenames[0],
					CacheDir:            cacheDir,
					LocalDir:            localDir,
					LocalDirUseSymlinks: localDirUseSymlinks,
					ForceDownload:       forceDownload,
					ResumeDownload:      resumeDownload,
					Token:               token,
					Quiet:               quiet,
				})
				if err != nil {
					return err
				}
				fmt.Println(outPath)
				return nil
			}

			outPath, err := client.SnapshotDownload(hf.SnapshotDownloadOptions{
				RepoID:              repoID,
				RepoType:            repoType,
				Revision:            revision,
				Filenames:           filenames,
				AllowPatterns:       include,
				IgnorePatterns:      exclude,
				CacheDir:            cacheDir,
				LocalDir:            localDir,
				LocalDirUseSymlinks: localDirUseSymlinks,
				ForceDownload:       forceDownload,
				ResumeDownload:      resumeDownload,
				Token:               token,
				Quiet:               quiet,
			})
			if err != nil {
				return err
			}
			fmt.Println(outPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoType, "repo-type", "model", "Type of repo to download from (defaults to 'model').")
	cmd.Flags().StringVar(&revision, "revision", "main", "An optional Git revision id which can be a branch name, a tag, or a commit hash.")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Glob patterns to match files to download.")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Glob patterns to exclude from files to download.")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Path to the directory where to save the downloaded files.")
	cmd.Flags().StringVar(&localDir, "local-dir", "", "If set, the downloaded file will be placed under this directory either as a symlink or a regular file.")
	cmd.Flags().StringVar(&localDirUseSymlinks, "local-dir-use-symlinks", "auto", "To be used with local_dir ('auto', 'True' or 'False').")
	cmd.Flags().BoolVar(&forceDownload, "force-download", false, "If True, the files will be downloaded even if they are already cached.")
	cmd.Flags().BoolVar(&resumeDownload, "resume-download", false, "If True, resume a previously interrupted download.")
	cmd.Flags().StringVar(&token, "token", "", "A User Access Token generated from https://huggingface.co/settings/tokens")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "If True, progress bars are disabled and only the path to the download files is printed.")

	return cmd
}
