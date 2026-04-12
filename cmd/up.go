package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/jmsperu/portfwd/config"
	"github.com/jmsperu/portfwd/forwarder"
	"github.com/spf13/cobra"
)

var allForwards bool

var upCmd = &cobra.Command{
	Use:   "up [name...]",
	Short: "Start saved port forwards",
	Long:  `Start one or more saved port forwards by name, or use --all to start all.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.Forwards) == 0 {
			fmt.Println("No saved forwards. Use 'portfwd add' to create one.")
			return
		}

		var names []string
		if allForwards {
			for name := range cfg.Forwards {
				names = append(names, name)
			}
		} else {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "Error: specify forward names or use --all")
				os.Exit(1)
			}
			names = args
		}

		var wg sync.WaitGroup
		var fwds []*forwarder.Forwarder

		for _, name := range names {
			entry, ok := cfg.Forwards[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "Warning: forward %q not found, skipping\n", name)
				continue
			}

			fwdCfg := forwarder.ForwardConfig{
				Name:       name,
				Protocol:   entry.Protocol,
				ListenPort: entry.ListenPort,
				RemoteHost: entry.RemoteHost,
				RemotePort: entry.RemotePort,
				TLSCert:    entry.TLSCert,
				TLSKey:     entry.TLSKey,
				AllowCIDRs: entry.AllowCIDRs,
				DenyCIDRs:  entry.DenyCIDRs,
				RateLimit:  entry.RateLimit,
			}

			fwd, err := forwarder.New(fwdCfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating forward %q: %v\n", name, err)
				continue
			}

			fwds = append(fwds, fwd)
			remote := fmt.Sprintf("%s:%d", entry.RemoteHost, entry.RemotePort)
			fmt.Printf("[%s] Forwarding %s :%d -> %s\n", name, strings.ToUpper(entry.Protocol), entry.ListenPort, remote)

			wg.Add(1)
			go func(f *forwarder.Forwarder) {
				defer wg.Done()
				if err := f.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}(fwd)
		}

		if len(fwds) == 0 {
			fmt.Println("No forwards started.")
			return
		}

		fmt.Println("Press Ctrl+C to stop all forwards")

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down...")
		for _, fwd := range fwds {
			fwd.Stop()
		}
	},
}

func init() {
	upCmd.Flags().BoolVarP(&allForwards, "all", "a", false, "Start all saved forwards")
	rootCmd.AddCommand(upCmd)
}
