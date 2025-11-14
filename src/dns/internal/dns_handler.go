
// dns_handler.go
// This module implements a minimal DNS server over UDP.
// It listens for DNS queries, parses the request, look up local records,
// and response with either the IP address or NXDOMAIN
// It handles basic A records and includes TTL in the responses. 

package internal

import (
    "bytes"
    "encoding/binary"
    "net"
    "strings"
	"fmt"
)

var dnslog = NewLogger("HEADER")

type Header struct {
    ID      uint16
    Flags   uint16
    QDCount uint16
    ANCount uint16
    NSCount uint16
    ARCount uint16
}


// StartUDP starts a UDP server to listen for DNS queries
func StartUDP(store *Store, listenAddr string) {
    addr, err := net.ResolveUDPAddr("udp", listenAddr)
    if err != nil {
        dnslog.Error("Failed to resolve UDP address %s: %v", listenAddr,err)
		return 
    }
    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        dnslog.Error("Failed to listen on UDP %s: %v", listenAddr, err)
		return
    }
    defer conn.Close()

	dnslog.Info("DNS UDP server listening on %s", listenAddr)

    buf := make([]byte, 512)
    for {
        n, clientAddr, err := conn.ReadFromUDP(buf)
        if err != nil {
			dnslog.Warn("Error reading from UDP: %v", err)
            continue
        }
        go handleQuery(conn, clientAddr, buf[:n], store)
    }
}


// handleQuery parses the query and sends a response
func handleQuery(conn *net.UDPConn, client *net.UDPAddr, data []byte, store *Store) {
    if len(data) < 12 {
		dnslog.Warn("Received invalid DNS packet (too short) from %s", client)
        return
    }
    id := binary.BigEndian.Uint16(data[0:2])
    qname, _ := parseQName(data[12:])
	lookupName := strings.TrimSuffix(strings.ToLower(qname), ".")
	
	dnslog.Info("Received query for %s from %s", lookupName,client)
    if rec, ok := store.Get(lookupName); ok {
        resp := buildResponse(id, qname, rec.IP)
        conn.WriteToUDP(resp, client)
		dnslog.Info("Response with IP %s for %s", rec.IP,qname)
    } else {
        conn.WriteToUDP(buildNX(id), client)
		dnslog.Info("Responded NXDOMAIN for %s", qname)
    }
}


// parseQname parses the domain name from DNs query format
func parseQName(data []byte) (string, int) {
    var labels []string
    i := 0
    for {
        l := int(data[i])
        if l == 0 {
            i++
            break
        }
        labels = append(labels, string(data[i+1:i+1+l]))
        i += l + 1
    }
    return strings.Join(labels, ".") + ".", i
}


// buildResponse constructs a minimal DNS response for A records
func buildResponse(id uint16, name, ip string) []byte {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.BigEndian, id)
    binary.Write(buf, binary.BigEndian, uint16(0x8180)) // Flags: respuesta sin error
    binary.Write(buf, binary.BigEndian, uint16(1))      // 1 pregunta
    binary.Write(buf, binary.BigEndian, uint16(1))      // 1 respuesta
    binary.Write(buf, binary.BigEndian, uint16(0))
    binary.Write(buf, binary.BigEndian, uint16(0))

    writeQName(buf, name)
    binary.Write(buf, binary.BigEndian, uint16(1)) // Tipo A
    binary.Write(buf, binary.BigEndian, uint16(1)) // Clase IN

    writeQName(buf, name)
    binary.Write(buf, binary.BigEndian, uint16(1))  // Tipo A
    binary.Write(buf, binary.BigEndian, uint16(1))  // Clase IN
    binary.Write(buf, binary.BigEndian, uint32(60)) // TTL
    binary.Write(buf, binary.BigEndian, uint16(4))  // Longitud RDATA
    for _, part := range strings.Split(ip, ".") {
        var p uint8
        fmt.Sscan(part, &p)
        buf.WriteByte(p)
    }
    return buf.Bytes()
}


// constructs a minimal NXDOMAIN response
func buildNX(id uint16) []byte {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.BigEndian, id)
    binary.Write(buf, binary.BigEndian, uint16(0x8183)) // NXDOMAIN
    binary.Write(buf, binary.BigEndian, uint16(1))
    binary.Write(buf, binary.BigEndian, uint16(0))
    binary.Write(buf, binary.BigEndian, uint16(0))
    binary.Write(buf, binary.BigEndian, uint16(0))
    return buf.Bytes()
}

// writes a DNS domain name in the proper format
func writeQName(buf *bytes.Buffer, name string) {
    for _, part := range strings.Split(strings.TrimSuffix(name, "."), ".") {
        buf.WriteByte(byte(len(part)))
        buf.WriteString(part)
    }
    buf.WriteByte(0)
}
