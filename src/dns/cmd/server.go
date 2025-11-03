package main

import (
    "log"
    "net"
    "src/dns"
)

// Simple record store
var records = map[string]string{
    "example.com.": "1.2.3.4",
	"domain.com.":"2.98.78.7",
	"server.com.":"89.78.67.0",
}

func addHostRecords(domain string, ip string){

    records[domain] = ip
}

func main() {
    addr := net.UDPAddr{
        Port: 8053,
        IP:   net.ParseIP("0.0.0.0"),
    }

    conn, err := net.ListenUDP("udp", &addr)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    log.Println("DNS server listening on UDP port 8053...")

    addHostRecords("web.","10.0.0.4")
    addHostRecords("database.","10.0.0.5")
    addHostRecords("host1.","10.0.0.6")

    buf := make([]byte, 512)
    for {
        n, clientAddr, err := conn.ReadFromUDP(buf)
        if err != nil {
            continue
        }

        domain, err := dns.ParseQuery(buf[:n])
        if err != nil {
            log.Println("parse error:", err)
            continue
        }

        log.Printf("Query for domain: %s", domain)

        ip, ok := records[domain]
        if !ok {
            log.Printf("No record for %s", domain)
            continue
        }

        response := dns.BuildResponse(buf[:n], ip)
        conn.WriteToUDP(response, clientAddr)
    }
}

