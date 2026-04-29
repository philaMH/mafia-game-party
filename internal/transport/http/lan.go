package httpx

import (
	"fmt"
	"io"
	"net"
)

// PrintLANAddresses writes one `http://<ip>:<port>` line per detected
// private IPv4 interface to w. The host's loopback and IPv6 addresses
// are intentionally skipped — only LAN-reachable IPs are useful for
// players opening the URL on their phones.
//
// On any net error or empty result, a single fallback line
// `http://localhost:<port>` is emitted so the host still has a usable
// link to copy.
func PrintLANAddresses(w io.Writer, port int) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Fprintf(w, "  (could not detect LAN: %v)\n", err)
		fmt.Fprintf(w, "  http://localhost:%d\n", port)
		return
	}

	found := 0
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP.To4()
		if ip == nil {
			continue
		}
		if ip.IsLoopback() {
			continue
		}
		if !ip.IsPrivate() {
			continue
		}
		fmt.Fprintf(w, "  http://%s:%d\n", ip.String(), port)
		found++
	}
	if found == 0 {
		fmt.Fprintf(w, "  http://localhost:%d\n", port)
	}
}
