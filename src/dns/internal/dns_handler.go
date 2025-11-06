package internal

import (
    "bytes"
    "encoding/binary"
    "log"
    "net"
    "strings"
	"fmt"
)

// Estructura DNS mínima: header + pregunta + respuesta
type Header struct {
    ID      uint16
    Flags   uint16
    QDCount uint16
    ANCount uint16
    NSCount uint16
    ARCount uint16
}

func StartUDP(store *Store, listenAddr string) {
    addr, err := net.ResolveUDPAddr("udp", listenAddr)
    if err != nil {
        log.Fatal(err)
    }
    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    buf := make([]byte, 512)
    for {
        n, clientAddr, err := conn.ReadFromUDP(buf)
        if err != nil {
            continue
        }
        go handleQuery(conn, clientAddr, buf[:n], store)
    }
}

func handleQuery(conn *net.UDPConn, client *net.UDPAddr, data []byte, store *Store) {
    if len(data) < 12 {
        return
    }
    id := binary.BigEndian.Uint16(data[0:2])
    qname, _ := parseQName(data[12:])

    if rec, ok := store.Get(strings.ToLower(qname)); ok {
        resp := buildResponse(id, qname, rec.IP)
        conn.WriteToUDP(resp, client)
    } else {
        conn.WriteToUDP(buildNX(id), client)
    }
}

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

func writeQName(buf *bytes.Buffer, name string) {
    for _, part := range strings.Split(strings.TrimSuffix(name, "."), ".") {
        buf.WriteByte(byte(len(part)))
        buf.WriteString(part)
    }
    buf.WriteByte(0)
}
