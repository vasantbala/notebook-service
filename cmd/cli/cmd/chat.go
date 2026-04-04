package cmd

import (
"bufio"
"fmt"
"os"
"strings"

"github.com/spf13/cobra"
"github.com/vasantbala/notebook-service/cmd/cli/internal/apiclient"
"github.com/vasantbala/notebook-service/cmd/cli/internal/cliconfig"
)

var chatCmd = &cobra.Command{
Use:   "chat",
Short: "Interactively chat with the active conversation (SSE streaming)",
RunE:  runChat,
}

func runChat(_ *cobra.Command, _ []string) error {
cfg, err := cliconfig.Load()
if err != nil {
return err
}
if cfg.URL == "" || cfg.Token == "" {
return fmt.Errorf("not logged in -- run: nbscli login")
}
if cfg.NotebookID == "" || cfg.ConversationID == "" {
return fmt.Errorf("no active conversation -- run: nbscli conversation")
}

client := apiclient.New(cfg.URL, cfg.Token)
reader := bufio.NewReader(os.Stdin)

fmt.Printf("Chatting in conversation %s\nType 'exit' or press Ctrl+C to quit.\n\n", cfg.ConversationID)

for {
fmt.Print("You: ")
query := readLine(reader)
if query == "" {
continue
}
if strings.EqualFold(query, "exit") || strings.EqualFold(query, "quit") {
fmt.Println("Goodbye!")
return nil
}

fmt.Print("Assistant: ")
err := client.ChatStream(
cfg.NotebookID,
cfg.ConversationID,
apiclient.ChatRequest{Query: query, TopK: 5},
func(token string) { fmt.Print(token) },
)
fmt.Println()
if err != nil {
fmt.Fprintf(os.Stderr, "error: %v\n", err)
}
}
}
