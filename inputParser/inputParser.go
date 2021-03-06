package inputParser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/octoman90/proxyshiva/proxy"
	"inet.af/netaddr"
)

func validateScanned(str string) bool {
	ipRegex := `((?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))`
	protocolRegex := `(http|https|socks4|socks5)`
	portRegex := `\d+`

	regex := regexp.MustCompile(`` +
		// Match protocol or protocol list
		`^(` + protocolRegex + `(,` + protocolRegex + `)*)+` +

		// Match "://"
		`:\/\/` +

		// Match IP or IP range
		ipRegex + `(-` + ipRegex + `)?` +

		// Match ":"
		`:` +

		// Match port or port range
		`(` + portRegex + `)(-` + portRegex + `)?$`,
	)

	return regex.MatchString(str)
}

func RequestGenerator(in string) chan proxy.Proxy {
	out := make(chan proxy.Proxy)

	if !validateScanned(in) {
		defer close(out)
		return out
	}

	schemeEndIndex := strings.Index(in, "://")
	addressStartIndex := schemeEndIndex + 3
	addressEndIndex := strings.LastIndex(in, ":")
	portStartIndex := addressEndIndex + 1

	// Parse schemes
	schemes := strings.Split(in[:schemeEndIndex], ",")

	// Parse the address range
	addressRange, _ := netaddr.ParseIPRange(in[addressStartIndex:addressEndIndex])
	if addressRange.String() == "zero IPRange" { // Handle single input
		addressRange.From, _ = netaddr.ParseIP(in[addressStartIndex:addressEndIndex])
		addressRange.To = addressRange.From
	}

	// Parse the port range
	var portRange [2]uint16
	if portRangeStr := strings.Split(in[portStartIndex:], "-"); len(portRangeStr) == 2 {
		sP, _ := strconv.Atoi(portRangeStr[0])
		eP, _ := strconv.Atoi(portRangeStr[1])

		portRange[0] = uint16(sP)
		portRange[1] = uint16(eP)
	} else {
		p, _ := strconv.Atoi(in[portStartIndex:])

		portRange[0] = uint16(p)
		portRange[1] = portRange[0]
	}

	go func() {
		defer close(out)

		for _, scheme := range schemes { // Rotate schemes
			for port := portRange[0]; port <= portRange[1]; port++ { // Rotate ports
				for address := addressRange.From; address.Less(addressRange.To) || address == addressRange.To; address = address.Next() { // Rotate IPs
					out <- proxy.Proxy{
						Scheme:  scheme,
						Address: address,
						Port:    port,
					}
				}
			}
		}
	}()

	return out
}
