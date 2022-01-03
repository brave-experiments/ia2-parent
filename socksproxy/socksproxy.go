package main

import (
	"context"
	"flag"
	"log"
	"net"

	socks5 "github.com/armon/go-socks5"
)

const (
	allowed = true
	denied  = false
)

type myRule struct {
	addrs []net.IP
	fqdns []string
}

func logConn(allowed bool, from, to *socks5.AddrSpec) {
	var prefix string
	if allowed {
		prefix = "Allowing"
	} else {
		prefix = "Denying"
	}
	log.Printf("%s connection request from %s:%d (%s) to %s:%d (%s).",
		prefix,
		from.IP, from.Port, from.FQDN,
		to.IP, to.Port, to.FQDN)
}

func (m myRule) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	for _, addr := range m.addrs {
		if req.DestAddr.IP.Equal(addr) {
			logConn(allowed, req.RemoteAddr, req.DestAddr)
			return ctx, allowed
		}
	}
	for _, fqdn := range m.fqdns {
		if req.DestAddr.FQDN == fqdn {
			logConn(allowed, req.RemoteAddr, req.DestAddr)
			return ctx, allowed
		}
	}
	logConn(denied, req.RemoteAddr, req.DestAddr)
	return ctx, denied
}

func main() {
	var addr string
	// allowedAddrs represents the list of IP addresses that the SOCKS server
	// allows connections to.  The list contains our Kafka cluster.
	allowedAddrs := []net.IP{}
	// allowedFQDNs represents the list of FQDNs that the SOCKS server allows
	// connections to.  The list contains Let's Encrypt's domain names.
	allowedFQDNs := []string{
		"acme-v02.api.letsencrypt.org",
	}

	flag.StringVar(&addr, "addr", ":1080", "Address to listen on.")
	flag.Parse()
	log.Printf("Starting SOCKSv5 server on %s.", addr)

	conf := &socks5.Config{
		Rules: myRule{
			addrs: allowedAddrs,
			fqdns: allowedFQDNs,
		},
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy.
	if err := server.ListenAndServe("tcp", addr); err != nil {
		panic(err)
	}
}
