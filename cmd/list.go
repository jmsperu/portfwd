package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jmsperu/portfwd/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List saved port forwards",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.Forwards) == 0 {
			fmt.Println("No saved forwards.")
			return
		}

		var names []string
		for name := range cfg.Forwards {
			names = append(names, name)
		}
		sort.Strings(names)

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROTOCOL\tLISTEN\tREMOTE\tACL\tRATE LIMIT")
		for _, name := range names {
			entry := cfg.Forwards[name]
			remote := fmt.Sprintf("%s:%d", entry.RemoteHost, entry.RemotePort)
			acl := "-"
			if len(entry.AllowCIDRs) > 0 {
				acl = "allow:" + strings.Join(entry.AllowCIDRs, ",")
			} else if len(entry.DenyCIDRs) > 0 {
				acl = "deny:" + strings.Join(entry.DenyCIDRs, ",")
			}
			rl := "-"
			if entry.RateLimit > 0 {
				rl = formatBytes(entry.RateLimit) + "/s"
			}
			fmt.Fprintf(w, "%s\t%s\t:%d\t%s\t%s\t%s\n",
				name, strings.ToUpper(entry.Protocol), entry.ListenPort, remote, acl, rl)
		}
		w.Flush()
	},
}

func formatBytes(b int) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
