package net

import "net"

// NetworkTarget represents network target resolved
type NetworkTarget struct {
	Typ    TargetType
	Raw    string
	Target interface{}
}

// ResolveTarget as NetworkTarget
func ResolveTarget(target string) *NetworkTarget {
	ipTarget, e := resolveIPTarget(target)
	if e != nil {
		return &NetworkTarget{
			Typ:    Unknown,
			Raw:    target,
			Target: e,
		}
	}
	return ipTarget
}

func resolveIPTarget(address string) (*NetworkTarget, error) {
	ipAddr, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return nil, err
	}
	return &NetworkTarget{
		Typ:    IP,
		Raw:    address,
		Target: ipAddr,
	}, nil
}
