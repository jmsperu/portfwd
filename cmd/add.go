package cmd

import (
	"fmt"
	"os"

	"github.com/jmsperu/portfwd/config"
	"github.com/jmsperu/portfwd/forwarder"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name> <local_port:remote_host:remote_port>",
	Short: "Save a port forward configuration",
	Long:  `Save a named port forward configuration to ~/.portfwd.yml for later use.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		spec, err := forwarder.ParseSpec(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		proto := "tcp"
		if udpMode {
			proto = "udp"
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fwd := config.ForwardEntry{
			Protocol:   proto,
			ListenPort: spec.ListenPort,
			RemoteHost: spec.RemoteHost,
			RemotePort: spec.RemotePort,
			TLSCert:    tlsCert,
			TLSKey:     tlsKey,
			RateLimit:  rateLimit,
		}

		if len(allowCIDRs) > 0 {
			fwd.AllowCIDRs = allowCIDRs
		}
		if len(denyCIDRs) > 0 {
			fwd.DenyCIDRs = denyCIDRs
		}

		cfg.Forwards[name] = fwd

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Saved forward %q: %s :%d -> %s:%d\n", name, proto, spec.ListenPort, spec.RemoteHost, spec.RemotePort)
	},
}

func init() {
	addCmd.Flags().BoolVarP(&udpMode, "udp", "u", false, "Use UDP instead of TCP")
	addCmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate file")
	addCmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS key file")
	addCmd.Flags().StringSliceVar(&allowCIDRs, "allow", nil, "Allowed source CIDRs")
	addCmd.Flags().StringSliceVar(&denyCIDRs, "deny", nil, "Denied source CIDRs")
	addCmd.Flags().IntVar(&rateLimit, "rate-limit", 0, "Rate limit in bytes/sec")
	rootCmd.AddCommand(addCmd)
}
