package driver

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/milosgajdos83/tenus"
)

const result = "%-10s %v\n"

type pool struct {
	ips     []net.IP
	pid     string
	value   string
	taken   bool
	gateway string
	bridge  *tenus.Bridger
	link    *tenus.Linker
}

func (p *pool) bridgeUp(verbose bool) error {
	//link, err := tenus.NewLink("l" + p.pid)
	link, err := tenus.NewLinkFrom("eth0")
	//link, err := tenus.NewLinkFrom("docker0")
	if err != nil {
		return err
	}

	//lIp, lIpNet, err := net.ParseCIDR(p.value)
	//if err != nil {
	//	return err
	//}
	//
	//if err := link.SetLinkIp(lIp, lIpNet); err != nil {
	//	return err
	//}

	//if verbose {
	//	fmt.Printf("Init bridge, ip: %v, ipNet: %v\n", lIp, lIpNet)
	//}

	//p.gateway = lIp.String()

	p.link = &link

	br, err := tenus.NewBridgeWithName(p.pid)
	//br, err := tenus.NewBridgeWithName()
	if err != nil {
		return err
	}

	brIp, brIpNet, err := net.ParseCIDR(p.value)
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("Init bridge, ip: %v, ipNet: %v\n", brIp, brIpNet)
	}

	if err := br.SetLinkIp(brIp, brIpNet); err != nil {
		return err
	}

	p.gateway = brIp.String()
	if err = br.SetLinkUp(); err != nil {
		return err
	}

	if err := br.AddSlaveIfc(link.NetInterface()); err != nil {
		return err
	}

	if err = link.SetLinkUp(); err != nil {
		return err
	}

	p.bridge = &br

	return nil
}

func makePool(verbose bool, value, pid string) (*pool, error) {
	ip, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		return nil, err
	}
	networkIP, broadcastIP, wildcardIP, networkMask := networkRange(ipnet)
	gateWay := networkIPInc(ip)

	if verbose {
		fmt.Printf(result, "Address:", ip)
		fmt.Printf(result, "Wildcard:", wildcardIP)
		fmt.Printf(result, "Network:", ipnet.IP)
		fmt.Printf(result, "NetworkIP:", networkIP)
		fmt.Printf(result, "Broadcast:", broadcastIP)
		fmt.Printf(result, "NetworkMask", networkMask)
		fmt.Printf(result, "Gateway", gateWay)
	}

	p := pool{
		value:   value,
		pid:     pid,
		taken:   false,
		gateway: gateWay.String() + "/32",
	}

	min := networkIPInc(networkIP)

	for {
		min = networkIPInc(min)

		if ipToInt(min) == ipToInt(broadcastIP) {
			break
		}

		p.ips = append(p.ips, min)
	}

	return &p, nil
}

// Calculates the first and last IP addresses in an IPNet
func networkRange(network *net.IPNet) (net.IP, net.IP, net.IP, net.IP) {
	netIP := network.IP.To4()
	networkIP := netIP.Mask(network.Mask)
	broadcastIP := net.IPv4(0, 0, 0, 0).To4()
	wildcardIP := net.IPv4(0, 0, 0, 0).To4()
	networkMask := net.IPv4(0, 0, 0, 0).To4()
	for i := 0; i < len(broadcastIP); i++ {
		broadcastIP[i] = netIP[i] | ^network.Mask[i]
		wildcardIP[i] = net.IPv4bcast[i] | ^network.Mask[i]
		networkMask[i] = ^wildcardIP[i]
	}
	return networkIP, broadcastIP, wildcardIP, networkMask
}

// Given a netmask, calculates the number of available hosts
//func networkSize(mask net.IPMask) int32 {
//	m := net.IPv4Mask(0, 0, 0, 0)
//	for i := 0; i < net.IPv4len; i++ {
//		m[i] = ^mask[i]
//	}
//	return int32(binary.BigEndian.Uint32(m)) + 1
//}

func networkIPInc(ip net.IP) net.IP {
	minIPNum := ipToInt(ip.To4()) + 1
	return intToIP(minIPNum)
}

//func networkIPDec(ip net.IP) net.IP {
//	maxIPNum := ipToInt(ip.To4()) - 1
//	return intToIP(maxIPNum)
//}

// Converts a 4 bytes IP into a 32 bit integer
func ipToInt(ip net.IP) int32 {
	return int32(binary.BigEndian.Uint32(ip.To4()))
}

// Converts 32 bit integer into a 4 bytes IP address
func intToIP(n int32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return net.IP(b)
}
