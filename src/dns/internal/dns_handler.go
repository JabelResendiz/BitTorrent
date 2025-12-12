package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

var dnslog = NewLogger("DNS")

// DNS constants
const (
	TypeA    = 1
	ClassIN  = 1
	FlagQR   = 1 << 15
	FlagAA   = 1 << 10
	FlagRD   = 1 << 8
	FlagRA   = 1 << 7
	RCODE_OK = 0
	RCODE_NX = 3
	RCODE_NI = 4 // Not Implemented
)

type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// ================================
// Start UDP Server
// ================================
func StartUDP(store *Store, listenAddr string) {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		dnslog.Error("Failed to resolve address %s: %v", listenAddr, err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		dnslog.Error("Failed to start UDP listener: %v", err)
		return
	}
	defer conn.Close()

	dnslog.Info("DNS UDP server listening on %s", listenAddr)

	buf := make([]byte, 512)

	for {
		n, client, err := conn.ReadFromUDP(buf)
		if err != nil {
			dnslog.Warn("Error reading UDP packet: %v", err)
			continue
		}

		go handleQuery(conn, client, buf[:n], store)
	}
}

// ================================
// Handle DNS Query
// ================================
func handleQuery(conn *net.UDPConn, client *net.UDPAddr, msg []byte, store *Store) {
	if len(msg) < 12 {
		dnslog.Warn("Received invalid DNS packet (too short) from %s", client)
		return
	}

	// Parse DNS header
	header := Header{
		ID:      binary.BigEndian.Uint16(msg[0:2]),
		Flags:   binary.BigEndian.Uint16(msg[2:4]),
		QDCount: binary.BigEndian.Uint16(msg[4:6]),
	}

	// Only support 1 question
	if header.QDCount != 1 {
		resp := buildErrorResponse(header.ID, RCODE_NI)
		conn.WriteToUDP(resp, client)
		dnslog.Warn("Received packet with QDCount=%d (unsupported)", header.QDCount)
		return
	}

	// Parse Question
	qname, offset := parseQName(msg[12:])
	qtype := binary.BigEndian.Uint16(msg[12+offset : 12+offset+2])
	qclass := binary.BigEndian.Uint16(msg[12+offset+2 : 12+offset+4])

	name := strings.TrimSuffix(strings.ToLower(qname), ".")

	dnslog.Info("Query: %s (type=%d) from %s", name, qtype, client)

	// Only support A + IN
	if qtype != TypeA || qclass != ClassIN {
		resp := buildErrorResponse(header.ID, RCODE_NI)
		conn.WriteToUDP(resp, client)
		dnslog.Warn("Unsupported query type=%d for %s", qtype, name)
		return
	}

	// Lookup in store
	rec, ok := store.Get(name)
	if !ok {
		resp := buildErrorResponse(header.ID, RCODE_NX)
		conn.WriteToUDP(resp, client)
		dnslog.Info("NXDOMAIN: %s", name)
		return
	}

	// Dynamic TTL
	remainingTTL := rec.TTL - int(time.Since(rec.Timestamp).Seconds())
	if remainingTTL <= 0 {
		resp := buildErrorResponse(header.ID, RCODE_NX)
		conn.WriteToUDP(resp, client)
		dnslog.Info("NXDOMAIN (TTL expired) for %s", name)
		return
	}

	// Build successful response
	// resp := buildAResponse(header.ID, qname, rec.IP, remainingTTL)
	if len(rec.IPs) == 0 {
		// no IPs → NXDOMAIN
		resp := buildErrorResponse(header.ID, RCODE_NX)
		conn.WriteToUDP(resp, client)
		return
	}

	ip := rec.IPs[0] // Round Robin handled in Store.Get()
	resp := buildAResponse(header.ID, qname, ip, remainingTTL)

	conn.WriteToUDP(resp, client)

	dnslog.Info("Response A %s → %s (TTL=%d)", name, ip, remainingTTL)
}

// ================================
// Protocol helpers
// ================================
func parseQName(data []byte) (string, int) {
	var labels []string
	i := 0
	for {
		if i >= len(data) {
			break
		}
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

func writeQName(buf *bytes.Buffer, name string) {
	for _, part := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		buf.WriteByte(byte(len(part)))
		buf.WriteString(part)
	}
	buf.WriteByte(0)
}

// ================================
// Response Builders
// ================================
func buildErrorResponse(id uint16, rcode uint16) []byte {
	buf := new(bytes.Buffer)

	flags := FlagQR | (rcode & 0xF)

	binary.Write(buf, binary.BigEndian, id)
	binary.Write(buf, binary.BigEndian, uint16(flags))
	binary.Write(buf, binary.BigEndian, uint16(1)) // QD
	binary.Write(buf, binary.BigEndian, uint16(0)) // AN
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))

	return buf.Bytes()
}

func buildAResponse(id uint16, name string, ip string, ttl int) []byte {
	buf := new(bytes.Buffer)

	flags := FlagQR | FlagAA | RCODE_OK

	// Header
	binary.Write(buf, binary.BigEndian, id)
	binary.Write(buf, binary.BigEndian, uint16(flags))
	binary.Write(buf, binary.BigEndian, uint16(1)) // QDCount
	binary.Write(buf, binary.BigEndian, uint16(1)) // ANCount
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))

	// Question
	writeQName(buf, name)
	binary.Write(buf, binary.BigEndian, uint16(TypeA))
	binary.Write(buf, binary.BigEndian, uint16(ClassIN))

	// Answer
	writeQName(buf, name)
	binary.Write(buf, binary.BigEndian, uint16(TypeA))
	binary.Write(buf, binary.BigEndian, uint16(ClassIN))
	binary.Write(buf, binary.BigEndian, uint32(ttl))
	binary.Write(buf, binary.BigEndian, uint16(4)) // IPv4 length

	for _, part := range strings.Split(ip, ".") {
		var b uint8
		fmt.Sscan(part, &b)
		buf.WriteByte(b)
	}

	return buf.Bytes()
}
