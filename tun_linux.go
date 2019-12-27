// +build linux

package gost

import (
	"fmt"
	"net"

	"github.com/docker/libcontainer/netlink"
	"github.com/go-log/log"
	"github.com/milosgajdos83/tenus"
	"github.com/songgao/water"
)

func createTun(cfg TunConfig) (conn net.Conn, ipNet *net.IPNet, err error) {
	ip, ipNet, err := net.ParseCIDR(cfg.Addr)
	if err != nil {
		return
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: cfg.Name,
		},
	})
	if err != nil {
		return
	}

	link, err := tenus.NewLinkFrom(ifce.Name())
	if err != nil {
		return
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = DefaultMTU
	}

	cmd := fmt.Sprintf("ip link set dev %s mtu %d", ifce.Name(), mtu)
	log.Log("[tun]", cmd)
	if er := link.SetLinkMTU(mtu); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	cmd = fmt.Sprintf("ip address add %s dev %s", cfg.Addr, ifce.Name())
	log.Log("[tun]", cmd)
	if er := link.SetLinkIp(ip, ipNet); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	cmd = fmt.Sprintf("ip link set dev %s up", ifce.Name())
	log.Log("[tun]", cmd)
	if er := link.SetLinkUp(); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	if err = addRoutes(ifce.Name(), cfg.Routes...); err != nil {
		return
	}

	conn = &tunConn{
		ifce: ifce,
		addr: &net.IPAddr{IP: ip},
	}
	return
}

func addRoutes(ifName string, routes ...string) error {
	for _, route := range routes {
		if route == "" {
			continue
		}
		cmd := fmt.Sprintf("ip route add %s dev %s", route, ifName)
		log.Log("[tun]", cmd)
		if err := netlink.AddRoute(route, "", "", ifName); err != nil {
			return fmt.Errorf("%s: %v", cmd, err)
		}
	}
	return nil
}
