package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Find out which huggingface.co account you are logged in as.",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := hf.GetToken()
			if err != nil || token == "" {
				fmt.Println("Not logged in")
				return nil
			}

			client := hf.NewClient("")
			info, err := client.Whoami(token)
			if err != nil {
				return err
			}

			fmt.Println(info.Name)
			if len(info.Orgs) > 0 {
				fmt.Printf("orgs:  %s\n", strings.Join(info.Orgs, ","))
			}
			if hf.Endpoint() != hf.DefaultEndpoint {
				fmt.Printf("Authenticated through private endpoint: %s\n", hf.Endpoint())
			}
			return nil
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := hf.GetToken()
			if token == "" {
				fmt.Println("Not logged in!")
				return nil
			}

			_ = hf.UnsetGitCredential()
			if err := hf.DeleteToken(); err != nil {
				return fmt.Errorf("failed deleting token: %w", err)
			}

			fmt.Println("Successfully logged out.")
			return nil
		},
	}
}
