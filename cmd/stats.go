package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jmsperu/portfwd/stats"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show traffic statistics for recent forwarding sessions",
	Run: func(cmd *cobra.Command, args []string) {
		entries, err := stats.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading stats: %v\n", err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Println("No statistics recorded yet. Run a forward to generate stats.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROTOCOL\tLISTEN\tREMOTE\tCONNECTIONS\tBYTES IN\tBYTES OUT\tLAST ACTIVE")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t:%d\t%s:%d\t%d\t%s\t%s\t%s\n",
				e.Name, e.Protocol, e.ListenPort,
				e.RemoteHost, e.RemotePort,
				e.Connections,
				formatBytes64(e.BytesIn), formatBytes64(e.BytesOut),
				e.LastActive.Format("2006-01-02 15:04:05"))
		}
		w.Flush()
	},
}

func formatBytes64(b int64) string {
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
	rootCmd.AddCommand(statsCmd)
}
