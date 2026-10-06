package hf

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// DumpEnvironmentInfo prints system and HF configuration information.
func DumpEnvironmentInfo(w io.Writer) {
	token, _ := GetToken()
	hasToken := token != ""

	who := "N/A"
	if hasToken {
		client := NewClient("")
		if info, err := client.Whoami(token); err == nil {
			who = info.Name
		}
	}

	helpers, _ := ListGitCredentialHelpers()
	helpersStr := strings.Join(helpers, ", ")
	if helpersStr == "" {
		helpersStr = "None"
	}

	fmt.Fprintln(w, "Copy-and-paste the text below in your GitHub issue.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- huggingface_hub version: %s (go)\n", Version)
	fmt.Fprintf(w, "- Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "- Go version: %s\n", runtime.Version())
	fmt.Fprintln(w, "- Running in iPython ?: No")
	fmt.Fprintln(w, "- Running in notebook ?: No")
	fmt.Fprintln(w, "- Running in Google Colab ?: No")
	fmt.Fprintf(w, "- Token path ?: %s\n", HFTokenPath())
	fmt.Fprintf(w, "- Has saved token ?: %t\n", hasToken)
	fmt.Fprintf(w, "- Who am I ?: %s\n", who)
	fmt.Fprintf(w, "- Configured git credential helpers: %s\n", helpersStr)
	fmt.Fprintf(w, "- ENDPOINT: %s\n", Endpoint())
	fmt.Fprintf(w, "- HF_HUB_CACHE: %s\n", HFHubCache())
	fmt.Fprintf(w, "- HF_ASSETS_CACHE: %s\n", HFAssetsCache())
	fmt.Fprintf(w, "- HF_TOKEN_PATH: %s\n", HFTokenPath())
	fmt.Fprintf(w, "- HF_HUB_OFFLINE: %t\n", HFHubOffline())
	fmt.Fprintf(w, "- HF_HUB_DISABLE_TELEMETRY: %s\n", os.Getenv("HF_HUB_DISABLE_TELEMETRY"))
	fmt.Fprintf(w, "- HF_HUB_DISABLE_PROGRESS_BARS: %t\n", HFHubDisableProgressBars())
	fmt.Fprintf(w, "- HF_HUB_ENABLE_HF_TRANSFER: %s\n", os.Getenv("HF_HUB_ENABLE_HF_TRANSFER"))
	fmt.Fprintln(w)
}
