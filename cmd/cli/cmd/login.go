package cmd

import (
"bufio"
"fmt"
"os"
"strings"

"github.com/spf13/cobra"
"github.com/vasantbala/notebook-service/cmd/cli/internal/cliconfig"
)

var loginCmd = &cobra.Command{
Use:   "login",
Short: "Save notebook-service URL and bearer token",
RunE:  runLogin,
}

var (
loginURL   string
loginToken string
)

func init() {
loginCmd.Flags().StringVar(&loginURL, "url", "", "Notebook-service base URL (e.g. http://localhost:8080)")
loginCmd.Flags().StringVar(&loginToken, "token", "", "Bearer token")
}

func runLogin(_ *cobra.Command, _ []string) error {
reader := bufio.NewReader(os.Stdin)

if loginURL == "" {
fmt.Print("Notebook-service URL: ")
url, _ := reader.ReadString('\n')
loginURL = strings.TrimSpace(url)
}

if loginToken == "" {
fmt.Print("Bearer token: ")
token, _ := reader.ReadString('\n')
loginToken = strings.TrimSpace(token)
}

if loginURL == "" || loginToken == "" {
return fmt.Errorf("url and token are required")
}

cfg, err := cliconfig.Load()
if err != nil {
return err
}
cfg.URL = loginURL
cfg.Token = loginToken
// Clear stale selection when credentials change.
cfg.NotebookID = ""
cfg.ConversationID = ""

if err := cliconfig.Save(cfg); err != nil {
return err
}
fmt.Println("Login saved.")
return nil
}
