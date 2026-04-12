package forwarder

import (
	"net"
)

type ACL struct {
	allowNets []*net.IPNet
	denyNets  []*net.IPNet
}

func NewACL(allow, deny []string) (*ACL, error) {
	acl := &ACL{}

	for _, cidr := range allow {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// Try as single IP
			ip := net.ParseIP(cidr)
			if ip == nil {
				return nil, err
			}
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			ipNet = &net.IPNet{IP: ip, Mask: mask}
		}
		acl.allowNets = append(acl.allowNets, ipNet)
	}

	for _, cidr := range deny {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			ip := net.ParseIP(cidr)
			if ip == nil {
				return nil, err
			}
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			ipNet = &net.IPNet{IP: ip, Mask: mask}
		}
		acl.denyNets = append(acl.denyNets, ipNet)
	}

	return acl, nil
}

func (a *ACL) Allowed(addr net.Addr) bool {
	if a == nil {
		return true
	}

	var ip net.IP
	switch v := addr.(type) {
	case *net.TCPAddr:
		ip = v.IP
	case *net.UDPAddr:
		ip = v.IP
	default:
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			return false
		}
		ip = net.ParseIP(host)
	}

	if ip == nil {
		return false
	}

	// Check deny list first
	for _, n := range a.denyNets {
		if n.Contains(ip) {
			return false
		}
	}

	// If allow list is set, IP must be in it
	if len(a.allowNets) > 0 {
		for _, n := range a.allowNets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	return true
}
