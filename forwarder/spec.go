package forwarder

import (
	"fmt"
	"strconv"
	"strings"
)

type Spec struct {
	ListenPort int
	RemoteHost string
	RemotePort int
}

// ParseSpec parses "local_port:remote_host:remote_port" format.
func ParseSpec(s string) (*Spec, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid spec %q: expected local_port:remote_host:remote_port", s)
	}

	listenPort, err := strconv.Atoi(parts[0])
	if err != nil || listenPort < 1 || listenPort > 65535 {
		return nil, fmt.Errorf("invalid listen port %q", parts[0])
	}

	remoteHost := parts[1]
	if remoteHost == "" {
		return nil, fmt.Errorf("remote host cannot be empty")
	}

	remotePort, err := strconv.Atoi(parts[2])
	if err != nil || remotePort < 1 || remotePort > 65535 {
		return nil, fmt.Errorf("invalid remote port %q", parts[2])
	}

	return &Spec{
		ListenPort: listenPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}, nil
}
