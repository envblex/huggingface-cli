package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newLoginCmd() *cobra.Command {
	var token string
	var addToGitCredential bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in using a token from huggingface.co/settings/tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			if token == "" {
				fmt.Print(`
    _|    _|  _|    _|    _|_|_|    _|_|_|  _|_|_|  _|      _|    _|_|_|      _|_|_|_|    _|_|      _|_|_|  _|_|_|_|
    _|    _|  _|    _|  _|        _|          _|    _|_|    _|  _|            _|        _|    _|  _|        _|
    _|_|_|_|  _|    _|  _|  _|_|  _|  _|_|    _|    _|  _|  _|  _|  _|_|      _|_|_|    _|_|_|_|  _|        _|_|_|
    _|    _|  _|    _|  _|    _|  _|    _|    _|    _|    _|_|  _|    _|      _|        _|    _|  _|        _|
    _|    _|    _|_|      _|_|_|    _|_|_|  _|_|_|  _|      _|    _|_|_|      _|        _|    _|    _|_|_|  _|_|_|_|
`)
				fmt.Print("To login, `huggingface_hub` requires a token generated from https://huggingface.co/settings/tokens .\nToken: ")
				t, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				token = strings.TrimSpace(t)

				fmt.Print("Add token as git credential? (Y/n) ")
				ans, _ := reader.ReadString('\n')
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans == "" || ans == "y" || ans == "yes" {
					addToGitCredential = true
				}
			}

			if token == "" {
				return fmt.Errorf("empty token provided")
			}

			client := hf.NewClient("")
			whoami, err := client.Whoami(token)
			if err != nil {
				return fmt.Errorf("invalid token passed: %w", err)
			}

			fmt.Printf("Token is valid (permission: %s).\n", whoami.Role)

			if addToGitCredential {
				if err := hf.SetGitCredential(token); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed setting git credentials: %v\n", err)
				} else {
					helpers, _ := hf.ListGitCredentialHelpers()
					helpersStr := strings.Join(helpers, ",")
					if helpersStr == "" {
						helpersStr = "store"
					}
					fmt.Printf("Your token has been saved in your configured git credential helpers (%s).\n", helpersStr)
				}
			} else {
				fmt.Println("Token has not been saved to git credential helper.")
			}

			if err := hf.SaveToken(token); err != nil {
				return fmt.Errorf("failed saving token: %w", err)
			}

			fmt.Printf("Your token has been saved to %s\n", hf.HFTokenPath())
			fmt.Println("Login successful")
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Token generated from https://huggingface.co/settings/tokens")
	cmd.Flags().BoolVar(&addToGitCredential, "add-to-git-credential", false, "Optional: Save token to git credential helper.")

	return cmd
}
