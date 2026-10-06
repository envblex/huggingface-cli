package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newRepoCmd() *cobra.Command {
	repoCmd := &cobra.Command{
		Use:   "repo",
		Short: "{create} Commands to interact with your huggingface.co repos.",
	}

	var repoType string
	var org string
	var spaceSDK string
	var yes bool

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new repo on huggingface.co",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			token, err := hf.GetToken()
			if err != nil || token == "" {
				return fmt.Errorf("not logged in. Run `huggingface-cli login` first")
			}

			client := hf.NewClient("")
			whoami, err := client.Whoami(token)
			if err != nil {
				return err
			}

			namespace := whoami.Name
			if org != "" {
				namespace = org
			}
			repoID := fmt.Sprintf("%s/%s", namespace, name)

			if !yes {
				fmt.Printf("You are about to create %s\nProceed? [Y/n] ", repoID)
				reader := bufio.NewReader(os.Stdin)
				ans, _ := reader.ReadString('\n')
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans != "" && ans != "y" && ans != "yes" {
					fmt.Println("Abort")
					return nil
				}
			}

			url, err := client.CreateRepo(hf.CreateRepoOptions{
				RepoID:       repoID,
				RepoType:     repoType,
				Organization: org,
				SpaceSDK:     spaceSDK,
				Token:        token,
			})
			if err != nil {
				return err
			}

			fmt.Println("\nYour repo now lives at:")
			fmt.Printf("  %s\n", url)
			fmt.Println("\nYou can clone it locally with the command below, and commit/push as usual.")
			fmt.Printf("\n  git clone %s\n\n", url)
			return nil
		},
	}

	createCmd.Flags().StringVar(&repoType, "type", "model", "Optional: repo_type: set to \"dataset\" or \"space\" if creating a dataset or space, default is model.")
	createCmd.Flags().StringVar(&org, "organization", "", "Optional: organization namespace.")
	createCmd.Flags().StringVar(&spaceSDK, "space_sdk", "", "Optional: Hugging Face Spaces SDK type. Required when --type is set to \"space\".")
	createCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Optional: answer Yes to the prompt")

	repoCmd.AddCommand(createCmd)
	return repoCmd
}
