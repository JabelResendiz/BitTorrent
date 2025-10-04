package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"src/bencode"
	"strings"
)

func main() {

	torrent, err := os.Open("test.torrent")

	if err != nil {
		panic(err)
	}

	meta, err := bencode.Decode(torrent)

	if err != nil {
		panic(err)
	}

	announce := meta["announce"].(string)
	info := meta["info"].(map[string]interface{})

	infoEncoded := bencode.Encode(info)
	infoHash := sha1.Sum(infoEncoded)

	var length int64
	if v, ok := info["length"].(int64); ok {
		length = v
	}

	var buf strings.Builder
	for _, b := range infoHash {
		buf.WriteString(fmt.Sprintf("%%%02X", b))
	}

	infoHashEncoded := buf.String()

	peerId := "-JC0001-123456789012"
	params := url.Values{
		"peer_id":    []string{peerId},
		"port":       []string{"6881"},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", length)},
		"compact":    []string{"1"},
		"event":      []string{"started"},
		"numwant":    []string{"50"},
		"key":        []string{"jc12345"},
	}

	fullURL := announce + "?info_hash=" + infoHashEncoded + "&" + params.Encode()
	fmt.Println("Tracker request: ", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	// decodificar la respuesta del tracker

	trackerResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		panic(err)
	}

	fmt.Println("Tracker responde: ", trackerResponse)

	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		data := []byte(peersRaw)

		for i := 0; i < len(data); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
			port := binary.BigEndian.Uint16(data[i+4 : i+6])
			fmt.Printf("Peer: %s:%d\n", ip, port)
		}
	}
}
