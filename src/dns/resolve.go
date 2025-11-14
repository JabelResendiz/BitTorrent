package dns


import (
	"context"
	"net"
	"time"
	"net/http"
)

func ResolveCustomHTTPClient(dnsAddr string) *http.Client {
    resolver := &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            d := net.Dialer{Timeout: time.Second}
            return d.DialContext(ctx, "udp", dnsAddr)
        },
    }

    dialer := &net.Dialer{
        Timeout:   5 * time.Second,
        Resolver:  resolver,
    }

    transport := &http.Transport{
        DialContext: dialer.DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
    }

    return &http.Client{
        Transport: transport,
        Timeout:   15 * time.Second,
    }
}