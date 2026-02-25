package VirtualRouterServer

import (
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sort"
	"strings"
)

type startupHosts struct {
	lan string
	wan string
}

func logHTTPAccessURLs(serviceName string, port int) {
	hosts := detectStartupHosts()

	slog.Info("服务访问地址", "service", serviceName, "scope", "localhost", "url", formatURL("http", "127.0.0.1", port))

	if hosts.lan != "" {
		slog.Info("服务访问地址", "service", serviceName, "scope", "lan", "url", formatURL("http", hosts.lan, port))
	} else {
		slog.Warn("服务访问地址未检测到内网 IP", "service", serviceName)
	}

	if hosts.wan != "" {
		slog.Info("服务访问地址", "service", serviceName, "scope", "wan", "url", formatURL("http", hosts.wan, port))
	} else {
		slog.Warn("服务访问地址未检测到公网 IP", "service", serviceName, "hint", "请使用服务器公网 IP 或端口映射地址")
	}
}

func logTCPAccessAddresses(serviceName string, port int) {
	hosts := detectStartupHosts()

	if hosts.lan != "" {
		slog.Info("服务连接地址", "service", serviceName, "scope", "lan", "address", formatHostPort(hosts.lan, port))
	} else {
		slog.Warn("服务连接地址未检测到内网 IP", "service", serviceName)
	}

	if hosts.wan != "" {
		slog.Info("服务连接地址", "service", serviceName, "scope", "wan", "address", formatHostPort(hosts.wan, port))
	} else {
		slog.Warn("服务连接地址未检测到公网 IP", "service", serviceName, "hint", "请使用服务器公网 IP 或端口映射地址")
	}
}

func detectStartupHosts() startupHosts {
	var lans []string
	var wans []string

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, addrErr := iface.Addrs()
			if addrErr != nil {
				continue
			}
			for _, addr := range addrs {
				ip := ipFromAddr(addr)
				if ip == nil || !ip.IsGlobalUnicast() {
					continue
				}
				normalized := ip.String()
				if parsed, ok := netip.AddrFromSlice(ip); ok {
					if parsed.IsPrivate() {
						lans = append(lans, normalized)
					} else {
						wans = append(wans, normalized)
					}
				}
			}
		}
	}

	lans = uniqueSorted(lans)
	wans = uniqueSorted(wans)

	hosts := startupHosts{}
	if len(lans) > 0 {
		hosts.lan = lans[0]
	}
	if len(wans) > 0 {
		hosts.wan = wans[0]
	}
	return hosts
}

func ipFromAddr(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func formatURL(scheme, host string, port int) string {
	return fmt.Sprintf("%s://%s", scheme, formatHostPort(host, port))
}

func formatHostPort(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}
