package cmd

import (
	"fmt"
	"os"

	"github.com/jmsperu/portfwd/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a saved port forward",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if _, ok := cfg.Forwards[name]; !ok {
			fmt.Fprintf(os.Stderr, "Error: forward %q not found\n", name)
			os.Exit(1)
		}

		delete(cfg.Forwards, name)

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Removed forward %q\n", name)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
