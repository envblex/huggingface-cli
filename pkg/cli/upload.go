package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newUploadCmd() *cobra.Command {
	var repoType string
	var revision string
	var private bool
	var include []string
	var exclude []string
	var deletePatterns []string
	var commitMessage string
	var commitDescription string
	var createPR bool
	var every float64
	var token string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "upload <repo_id> [<local_path>] [<path_in_repo>]",
		Short: "Upload a file or a folder to a repo on the Hub",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoID := args[0]
			var localPath string
			var pathInRepo string

			if len(args) > 1 {
				localPath = args[1]
			}
			if len(args) > 2 {
				pathInRepo = args[2]
			}

			// Implicit resolution from python upload.py:
			if localPath == "" {
				repoName := repoID
				if idx := len(repoID) - 1; idx >= 0 {
					parts := splitSlash(repoID)
					repoName = parts[len(parts)-1]
				}
				if fi, err := os.Stat(repoName); err == nil {
					localPath = repoName
					if fi.IsDir() {
						pathInRepo = "."
					} else {
						pathInRepo = repoName
					}
				} else {
					return fmt.Errorf("'%s' is not a local file or folder. Please set `local_path` explicitly", repoName)
				}
			} else if pathInRepo == "" {
				if fi, err := os.Stat(localPath); err == nil && !fi.IsDir() {
					pathInRepo = ""
				} else {
					pathInRepo = "."
				}
			}

			client := hf.NewClient("")
			url, err := client.Upload(hf.UploadOptions{
				RepoID:            repoID,
				LocalPath:         localPath,
				PathInRepo:        pathInRepo,
				RepoType:          repoType,
				Revision:          revision,
				Private:           private,
				Include:           include,
				Exclude:           exclude,
				Delete:            deletePatterns,
				CommitMessage:     commitMessage,
				CommitDescription: commitDescription,
				CreatePR:          createPR,
				Every:             every,
				Token:             token,
				Quiet:             quiet,
			})
			if err != nil {
				return err
			}

			fmt.Println(url)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoType, "repo-type", "model", "Type of repo to upload to (defaults to 'model').")
	cmd.Flags().StringVar(&revision, "revision", "main", "An optional Git revision id which can be a branch name or tag.")
	cmd.Flags().BoolVar(&private, "private", false, "Whether the repo should be private.")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Glob patterns to match files to upload.")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Glob patterns to exclude from files to upload.")
	cmd.Flags().StringSliceVar(&deletePatterns, "delete", nil, "Glob patterns of files to delete from the repo.")
	cmd.Flags().StringVar(&commitMessage, "commit-message", "", "The summary / title of the commit.")
	cmd.Flags().StringVar(&commitDescription, "commit-description", "", "The description of the commit.")
	cmd.Flags().BoolVar(&createPR, "create-pr", false, "Whether to create a Pull Request.")
	cmd.Flags().Float64Var(&every, "every", 0, "Upload at regular intervals (in minutes).")
	cmd.Flags().StringVar(&token, "token", "", "A User Access Token generated from https://huggingface.co/settings/tokens")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Disable warnings and progress bars.")

	return cmd
}

func splitSlash(s string) []string {
	var res []string
	cur := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			if cur != "" {
				res = append(res, cur)
				cur = ""
			}
		} else {
			cur += string(s[i])
		}
	}
	if cur != "" {
		res = append(res, cur)
	}
	return res
}
