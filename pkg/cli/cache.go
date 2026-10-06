package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/envblex/huggingface-cli/pkg/hf"
)

func newScanCacheCmd() *cobra.Command {
	var dir string
	var verboseCount int

	cmd := &cobra.Command{
		Use:   "scan-cache",
		Short: "Scan cache directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			t0 := time.Now()
			cacheInfo, err := hf.ScanCacheDir(dir)
			if err != nil {
				return err
			}
			elapsed := time.Since(t0)

			table := hf.FormatCacheTable(cacheInfo, verboseCount)
			fmt.Println(table)

			fmt.Printf("\nDone in %.1fs. Scanned %d repo(s) for a total of %s.\n",
				elapsed.Seconds(), len(cacheInfo.Repos), hf.FormatSize(cacheInfo.SizeOnDisk))

			if len(cacheInfo.Warnings) > 0 {
				if verboseCount >= 3 {
					fmt.Printf("Got %d warning(s) while scanning:\n", len(cacheInfo.Warnings))
					for _, w := range cacheInfo.Warnings {
						fmt.Println(" ", w)
					}
				} else {
					fmt.Printf("Got %d warning(s) while scanning. Use -vvv to print details.\n", len(cacheInfo.Warnings))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Cache directory to scan (optional). Default to default HuggingFace cache.")
	cmd.Flags().CountVarP(&verboseCount, "verbose", "v", "Show a more verbose output (-v for revisions, -vvv for warnings).")

	return cmd
}

func newDeleteCacheCmd() *cobra.Command {
	var dir string
	var disableTUI bool

	cmd := &cobra.Command{
		Use:   "delete-cache",
		Short: "Delete revisions from the cache directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheInfo, err := hf.ScanCacheDir(dir)
			if err != nil {
				return err
			}

			if len(cacheInfo.Repos) == 0 {
				fmt.Println("Cache directory is empty. Nothing to delete.")
				return nil
			}

			type revChoice struct {
				index        int
				repoID       string
				commitHash   string
				sizeStr      string
				refs         string
				snapshotPath string
			}

			var choices []revChoice
			idx := 1
			for _, repo := range cacheInfo.Repos {
				for _, rev := range repo.Revisions {
					choices = append(choices, revChoice{
						index:        idx,
						repoID:       repo.RepoID,
						commitHash:   rev.CommitHash,
						sizeStr:      hf.FormatSize(rev.SizeOnDisk),
						refs:         strings.Join(rev.Refs, ", "),
						snapshotPath: rev.SnapshotPath,
					})
					idx++
				}
			}

			if len(choices) == 0 {
				fmt.Println("No revisions found in cache.")
				return nil
			}

			fmt.Println("\nAvailable revisions in cache:")
			for _, c := range choices {
				refStr := ""
				if c.refs != "" {
					refStr = fmt.Sprintf(" (%s)", c.refs)
				}
				fmt.Printf("[%2d] %-30s %s  %s%s\n", c.index, c.repoID, c.commitHash[:10], c.sizeStr, refStr)
			}

			fmt.Print("\nEnter numbers of revisions to delete (e.g. '1, 2', 'all', or 'q' to cancel): ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			input = strings.TrimSpace(input)
			if input == "q" || input == "quit" || input == "" {
				fmt.Println("Deletion is cancelled. Do nothing.")
				return nil
			}

			var selectedHashes []string
			if strings.ToLower(input) == "all" {
				for _, c := range choices {
					selectedHashes = append(selectedHashes, c.commitHash)
				}
			} else {
				parts := strings.Split(input, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if n, err := strconv.Atoi(p); err == nil && n >= 1 && n <= len(choices) {
						selectedHashes = append(selectedHashes, choices[n-1].commitHash)
					} else {
						// Check if full/short hash entered
						for _, c := range choices {
							if strings.HasPrefix(c.commitHash, p) {
								selectedHashes = append(selectedHashes, c.commitHash)
							}
						}
					}
				}
			}

			if len(selectedHashes) == 0 {
				fmt.Println("No revisions selected. Deletion cancelled.")
				return nil
			}

			strategy, err := cacheInfo.DeleteRevisions(selectedHashes)
			if err != nil {
				return err
			}

			fmt.Printf("\nSelected %d revision(s). Will free %s.\n", len(selectedHashes), strategy.ExpectedFreedStr)
			fmt.Print("Confirm deletion ? (y/N): ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.ToLower(strings.TrimSpace(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Deletion is cancelled. Do nothing.")
				return nil
			}

			fmt.Println("Start deletion.")
			if err := strategy.Execute(); err != nil {
				return fmt.Errorf("deletion failed: %w", err)
			}

			fmt.Printf("Done. Deleted %d repo(s) and %d revision(s) for a total of %s.\n",
				len(strategy.Repos), len(strategy.Snapshots), strategy.ExpectedFreedStr)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Cache directory (optional). Default to default HuggingFace cache.")
	cmd.Flags().BoolVar(&disableTUI, "disable-tui", false, "Disable Terminal User Interface mode.")

	return cmd
}
