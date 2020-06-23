package ping

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func Ping(host string, times uint, timeout time.Duration) ([]RoundTrip, error) {
	ipaddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, err
	}
	var conn *icmp.PacketConn
	isIPv4 := ipaddr.IP.To4() != nil
	// Support socket ICMP only currently
	if isIPv4 {
		conn, err = listen("ip4:icmp")
		if err != nil {
			return nil, err
		}
		conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	} else {
		conn, err = listen("ip6:icmp")
		if err != nil {
			return nil, err
		}
		conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}
	defer conn.Close()
	return newPiper(*ipaddr, conn, isIPv4, times, timeout, 64).Run(), nil
}

func listen(network string) (*icmp.PacketConn, error) {
	// empty address is a hint wildcard for Go 1.0 undocumented behavior
	conn, err := icmp.ListenPacket(network, "")
	if err != nil {
		return nil, err
	}
	return conn, nil
}
