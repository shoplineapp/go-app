package common

import (
	"net"
	"os"
)

var (
	instanceIp *string = nil
)

func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return GetInstanceIP()
	}
	return hostname
}

func GetInstanceIP() string {
	if instanceIp != nil {
		return *instanceIp
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.IsLoopback() {
				continue
			}
			ipv4 := ipNet.IP.To4()
			if ipv4 != nil {
				ip := ipv4.String()
				instanceIp = &ip
				return ip
			}
		}
	}
	instanceIp = new(string)
	return ""
}
