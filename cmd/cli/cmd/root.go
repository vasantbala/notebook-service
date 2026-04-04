package cmd

import (
"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
Use:   "nbscli",
Short: "Notebook Service CLI -- chat with your notebooks from the terminal",
}

// Execute is called by main.
func Execute() error {
return rootCmd.Execute()
}

func init() {
rootCmd.AddCommand(loginCmd)
rootCmd.AddCommand(conversationCmd)
rootCmd.AddCommand(chatCmd)
}
