package cmd

import (
"bufio"
"fmt"
"os"
"strconv"
"strings"

"github.com/spf13/cobra"
"github.com/vasantbala/notebook-service/cmd/cli/internal/apiclient"
"github.com/vasantbala/notebook-service/cmd/cli/internal/cliconfig"
"github.com/vasantbala/notebook-service/internal/model"
)

var conversationCmd = &cobra.Command{
Use:   "conversation",
Short: "Select a notebook and pick or create a conversation",
RunE:  runConversation,
}

func runConversation(_ *cobra.Command, _ []string) error {
cfg, err := cliconfig.Load()
if err != nil {
return err
}
if cfg.URL == "" || cfg.Token == "" {
return fmt.Errorf("not logged in -- run: nbscli login")
}

client := apiclient.New(cfg.URL, cfg.Token)
reader := bufio.NewReader(os.Stdin)

notebookID, err := pickNotebook(client, reader, cfg.NotebookID)
if err != nil {
return err
}
cfg.NotebookID = notebookID

conversationID, err := pickOrCreateConversation(client, reader, notebookID)
if err != nil {
return err
}
cfg.ConversationID = conversationID

if err := cliconfig.Save(cfg); err != nil {
return err
}
fmt.Println("\nActive conversation set. Run 'nbscli chat' to start chatting.")
return nil
}

// pickNotebook lists notebooks and returns the ID of the user-selected one.
func pickNotebook(client *apiclient.Client, reader *bufio.Reader, currentID string) (string, error) {
notebooks, err := client.ListNotebooks()
if err != nil {
return "", fmt.Errorf("list notebooks: %w", err)
}
if len(notebooks) == 0 {
return "", fmt.Errorf("no notebooks found -- create one via the API first")
}

fmt.Println("\nNotebooks:")
for i, nb := range notebooks {
marker := " "
if nb.ID == currentID {
marker = "*"
}
fmt.Printf("  %s[%d] %s\n", marker, i+1, notebookLabel(nb))
}
fmt.Print("Select notebook [1]: ")

line := readLine(reader)
if line == "" {
line = "1"
}
idx, err := strconv.Atoi(line)
if err != nil || idx < 1 || idx > len(notebooks) {
return "", fmt.Errorf("invalid selection")
}
selected := notebooks[idx-1]
fmt.Printf("Notebook: %s\n", selected.Title)
return selected.ID, nil
}

// pickOrCreateConversation lists conversations and returns the chosen/created ID.
func pickOrCreateConversation(client *apiclient.Client, reader *bufio.Reader, notebookID string) (string, error) {
convs, err := client.ListConversations(notebookID)
if err != nil {
return "", fmt.Errorf("list conversations: %w", err)
}

fmt.Println("\nConversations:")
fmt.Println("  [0] Create new conversation")
for i, c := range convs {
fmt.Printf("  [%d] %s\n", i+1, conversationLabel(c))
}
fmt.Print("Select [0]: ")

line := readLine(reader)
if line == "" {
line = "0"
}
idx, err := strconv.Atoi(line)
if err != nil || idx < 0 || idx > len(convs) {
return "", fmt.Errorf("invalid selection")
}
if idx == 0 {
return createNewConversation(client, reader, notebookID)
}
selected := convs[idx-1]
fmt.Printf("Conversation: %s\n", selected.Title)
return selected.ID, nil
}

func createNewConversation(client *apiclient.Client, reader *bufio.Reader, notebookID string) (string, error) {
fmt.Print("Conversation title: ")
title := readLine(reader)
if title == "" {
title = "New Conversation"
}
conv, err := client.CreateConversation(notebookID, title)
if err != nil {
return "", fmt.Errorf("create conversation: %w", err)
}
fmt.Printf("Created: %s\n", conv.Title)
return conv.ID, nil
}

func notebookLabel(nb model.Notebook) string {
if nb.Description != "" {
return fmt.Sprintf("%s -- %s", nb.Title, nb.Description)
}
return nb.Title
}

func conversationLabel(c model.Conversation) string {
return fmt.Sprintf("%s  (%s)", c.Title, c.CreatedAt.Format("2006-01-02"))
}

func readLine(reader *bufio.Reader) string {
line, _ := reader.ReadString('\n')
return strings.TrimSpace(line)
}
