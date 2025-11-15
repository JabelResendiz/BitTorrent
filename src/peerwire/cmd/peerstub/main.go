package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"src/bencode"
)

const (
	pstr    = "BitTorrent protocol"
	pstrlen = 19

	MsgChoke      = 0
	MsgUnchoke    = 1
	MsgInterested = 2
	MsgHave       = 4
	MsgBitfield   = 5
	MsgRequest    = 6
	MsgPiece      = 7
)

func percentEncode20(b [20]byte) string {
	var sb strings.Builder
	for _, x := range b {
		sb.WriteString(fmt.Sprintf("%%%02X", x))
	}
	return sb.String()
}

func announceTracker(announce string, infoHash [20]byte, peerID string, port int, left int64) {
	q := url.Values{
		"peer_id":    []string{peerID},
		"port":       []string{fmt.Sprintf("%d", port)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", left)},
		"compact":    []string{"1"},
		"event":      []string{"started"},
		"numwant":    []string{"50"},
		"key":        []string{"stub"},
		"ip":         []string{"127.0.0.1"},
	}
	fullURL := announce + "?info_hash=" + percentEncode20(infoHash) + "&" + q.Encode()
	resp, err := http.Get(fullURL)
	if err != nil {
		log.Printf("announce error: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	log.Printf("announced to tracker: %s", fullURL)
}

func writeMessage(w io.Writer, id byte, payload []byte) error {
	length := uint32(1 + len(payload))
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, length); err != nil {
		return err
	}
	buf.WriteByte(id)
	if len(payload) > 0 {
		buf.Write(payload)
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func handleConn(c net.Conn, infoHash [20]byte, peerID [20]byte, pieceLen int, totalLen int64) {
	defer c.Close()

	// Read client's handshake
	hs := make([]byte, 49+pstrlen)
	if _, err := io.ReadFull(c, hs); err != nil {
		log.Printf("handshake read error: %v", err)
		return
	}
	if int(hs[0]) != pstrlen || string(hs[1:20]) != pstr {
		log.Printf("invalid handshake pstr")
		return
	}
	if !bytes.Equal(hs[28:48], infoHash[:]) {
		log.Printf("info_hash mismatch")
		return
	}
	// Send our handshake
	out := new(bytes.Buffer)
	out.WriteByte(pstrlen)
	out.WriteString(pstr)
	out.Write(make([]byte, 8))
	out.Write(infoHash[:])
	out.Write(peerID[:])
	if _, err := c.Write(out.Bytes()); err != nil {
		log.Printf("handshake write error: %v", err)
		return
	}
	log.Printf("handshake OK with %s", c.RemoteAddr())

	// Send bitfield (we claim to have piece 0)
	// Determine number of pieces (ceil(totalLen/pieceLen))
	numPieces := int((totalLen + int64(pieceLen) - 1) / int64(pieceLen))
	bf := make([]byte, (numPieces+7)/8)
	if numPieces > 0 {
		bf[0] = 0x80
	} // set piece 0
	if err := writeMessage(c, MsgBitfield, bf); err != nil {
		log.Printf("send bitfield: %v", err)
	}
	// Send unchoke
	if err := writeMessage(c, MsgUnchoke, nil); err != nil {
		log.Printf("send unchoke: %v", err)
	}

	// Read loop for requests
	for {
		var length uint32
		if err := binary.Read(c, binary.BigEndian, &length); err != nil {
			log.Printf("read len: %v", err)
			return
		}
		if length == 0 {
			continue
		} // keep-alive
		data := make([]byte, length)
		if _, err := io.ReadFull(c, data); err != nil {
			log.Printf("read msg: %v", err)
			return
		}
		id := data[0]
		payload := data[1:]
		switch id {
		case MsgInterested:
			// ignore
		case MsgBitfield:
			// pequeño log para ver el bitfield que envía el cliente
			cnt := 0
			for _, b := range payload {
				for i := 0; i < 8; i++ {
					if b&(1<<uint(7-i)) != 0 {
						cnt++
					}
				}
			}
			log.Printf("stub recibió BITFIELD del cliente: len=%d bytes, piezas=%d, hex=%x", len(payload), cnt, payload)
		case MsgRequest:
			if len(payload) != 12 {
				continue
			}
			idx := binary.BigEndian.Uint32(payload[0:4])
			begin := binary.BigEndian.Uint32(payload[4:8])
			reqLen := binary.BigEndian.Uint32(payload[8:12])
			if idx != 0 {
				continue
			}
			// cap to piece size
			psize := totalLen // single piece scenario ok; general: last piece size
			if int64(begin) >= psize {
				continue
			}
			max := psize - int64(begin)
			L := int64(reqLen)
			if L > max {
				L = max
			}
			if L < 0 {
				L = 0
			}
			block := make([]byte, L)
			// generate deterministic content (zeros is fine)
			// Send piece
			out := new(bytes.Buffer)
			// length = 9 + len(block)
			total := uint32(9 + len(block))
			binary.Write(out, binary.BigEndian, total)
			out.WriteByte(MsgPiece)
			binary.Write(out, binary.BigEndian, idx)
			binary.Write(out, binary.BigEndian, begin)
			out.Write(block)
			if _, err := c.Write(out.Bytes()); err != nil {
				log.Printf("write piece: %v", err)
				return
			}
			log.Printf("sent piece idx=%d begin=%d len=%d", idx, begin, len(block))
		case MsgHave:
			if len(payload) >= 4 {
				idx := binary.BigEndian.Uint32(payload[0:4])
				log.Printf("stub recibió HAVE de pieza %d", idx)
			} else {
				log.Printf("stub recibió HAVE inválido (payload corto)")
			}
		default:
			// ignore others
		}
	}
}

func main() {
	// Read torrent to get info_hash and metadata
	f, err := os.Open("test.torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	meta, err := bencode.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	info := meta["info"].(map[string]interface{})
	announce := meta["announce"].(string)
	infoEncoded := bencode.Encode(info)
	infoHash := sha1.Sum(infoEncoded)
	// lengths
	var total int64
	if v, ok := info["length"].(int64); ok {
		total = v
	}
	var pieceLen int64
	if v, ok := info["piece length"].(int64); ok {
		pieceLen = v
	}
	if pieceLen <= 0 {
		pieceLen = 16384
	}

	// Prepare IDs
	var peerID [20]byte
	copy(peerID[:], []byte("-STUB-PEER-0000000000")[:20])

	// Announce to tracker as seeder (left=0) on port 6881
	announceTracker(announce, infoHash, string(peerID[:]), 6881, 0)

	// Listen on 6881
	ln, err := net.Listen("tcp", ":6881")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("peer stub listening on :6881")

	// Accept 1 connection at a time (simple)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(conn, infoHash, peerID, int(pieceLen), total)
		// avoid busy loop
		time.Sleep(10 * time.Millisecond)
	}
}
