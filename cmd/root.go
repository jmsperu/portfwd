package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmsperu/portfwd/forwarder"
	"github.com/spf13/cobra"
)

var (
	udpMode    bool
	tlsCert    string
	tlsKey     string
	allowCIDRs []string
	denyCIDRs  []string
	rateLimit  int
)

var rootCmd = &cobra.Command{
	Use:   "portfwd [local_port:remote_host:remote_port]",
	Short: "TCP/UDP port forwarder with saved configs, traffic stats, and access control",
	Long: `portfwd is a cross-platform TCP/UDP port forwarder.

Forward local ports to remote destinations without SSH. Supports saved
configurations, traffic statistics, connection logging, rate limiting,
access control, and optional TLS termination.

Examples:
  portfwd 8080:remote.host:80             # forward TCP port immediately
  portfwd -u 5060:sip.server:5060         # forward UDP port immediately
  portfwd add web 8080:remote.host:80     # save a forward config
  portfwd up web                           # start a saved forward
  portfwd up --all                         # start all saved forwards
  portfwd list                             # list saved forwards
  portfwd stats                            # show traffic statistics
  portfwd remove web                       # remove a saved forward`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		spec, err := forwarder.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		proto := "tcp"
		if udpMode {
			proto = "udp"
		}

		cfg := forwarder.ForwardConfig{
			Name:       "adhoc",
			Protocol:   proto,
			ListenPort: spec.ListenPort,
			RemoteHost: spec.RemoteHost,
			RemotePort: spec.RemotePort,
			TLSCert:    tlsCert,
			TLSKey:     tlsKey,
			RateLimit:  rateLimit,
		}

		if len(allowCIDRs) > 0 {
			cfg.AllowCIDRs = allowCIDRs
		}
		if len(denyCIDRs) > 0 {
			cfg.DenyCIDRs = denyCIDRs
		}

		fwd, err := forwarder.New(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		remote := fmt.Sprintf("%s:%d", spec.RemoteHost, spec.RemotePort)
		fmt.Printf("Forwarding %s :%d -> %s\n", strings.ToUpper(proto), spec.ListenPort, remote)
		fmt.Println("Press Ctrl+C to stop")

		if err := fwd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&udpMode, "udp", "u", false, "Use UDP instead of TCP")
	rootCmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate file for TLS termination")
	rootCmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS key file for TLS termination")
	rootCmd.Flags().StringSliceVar(&allowCIDRs, "allow", nil, "Allowed source CIDRs (comma-separated)")
	rootCmd.Flags().StringSliceVar(&denyCIDRs, "deny", nil, "Denied source CIDRs (comma-separated)")
	rootCmd.Flags().IntVar(&rateLimit, "rate-limit", 0, "Rate limit in bytes/sec per connection (0 = unlimited)")
}
