package uri

import (
	"strconv"
	"strings"
)

// IsDefaultPort indicates if the port is specified and is different from
// the defaut port defined for this scheme (if any).
//
// For example, an URI like http://host:8080 would return false, since 80 is the default http port.
func (u uri) IsDefaultPort() bool {
	if len(u.authority.port) == 0 {
		return true
	}

	portNum, _ := strconv.ParseUint(u.authority.port, 10, 64)

	return defaultPortForScheme(strings.ToLower(u.scheme)) == portNum
}

// DefaultPort returns the default standardized port for the scheme of this URI,
// or zero if no such default is known.
//
// For example, for scheme "https", the default port is 443.
func (u uri) DefaultPort() int {
	return int(defaultPortForScheme(strings.ToLower(u.scheme))) //nolint:gosec // uint64 -> int conversion is ok: no port overflows a int
}

// References:
// * https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
// * https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml
//
// Also: https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
func defaultPortForScheme(scheme string) uint64 {
	//nolint:mnd // no need to define default ports with additional constants
	switch scheme {
	case "aaa":
		return 3868
	case "aaas":
		return 5658
	case "acap":
		return 674
	case "cap":
		return 1026
	case "coap", "coap+tcp":
		return 5683
	case "coaps":
		return 5684
	case "coap+ws":
		return 80
	case "coaps+ws":
		return 443
	case "dict":
		return 2628
	case "dns":
		return 53
	case "finger":
		return 79
	case "ftp":
		return 21
	case "git":
		return 9418
	case "go":
		return 1096
	case "gopher":
		return 70
	case "http":
		return 80
	case "https":
		return 443
	case "iax":
		return 4569
	case "icap":
		return 1344
	case "imap":
		return 143
	case "ipp", "ipps":
		return 631
	case "irc", "irc6":
		return 6667
	case "ircs":
		return 6697
	case "ldap":
		return 389
	case "mailto":
		return 25
	case "msrp", "msrps":
		return 2855
	case "nfs":
		return 2049
	case "nntp":
		return 119
	case "ntp":
		return 123
	case "postgresql":
		return 5432
	case "radius":
		return 1812
	case "redis":
		return 6379
	case "rmi":
		return 1098
	case "rtsp", "rtsps", "rtspu":
		return 554
	case "rsync":
		return 873
	case "sftp":
		return 22
	case "skype":
		return 23399
	case "smtp":
		return 25
	case "snmp":
		return 161
	case "ssh":
		return 22
	case "steam":
		return 7777
	case "svn":
		return 3690
	case "telnet":
		return 23
	case "vnc":
		return 5500
	case "wss":
		return 6602
	}

	return 0
}
