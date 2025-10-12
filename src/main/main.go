package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"src/bencode"
)

func main() {
	// For reproducible tests, we'll define content bytes (all zeros)
	totalLen := int64(12345)
	pieceLen := int64(16384)
	// Single piece (since totalLen < pieceLen)
	content := make([]byte, totalLen) // zero-filled
	sum := sha1.Sum(content)

	torrent := map[string]interface{}{
		// Apunta al tracker local por defecto
		"announce": "http://localhost:8080/announce",
		"info": map[string]interface{}{
			"name":         "archivo.txt",
			"length":       totalLen,
			"piece length": pieceLen,
			// 'pieces' is binary string of 20-byte SHA1(s)
			"pieces": string(sum[:]),
		},
	}

	data := bencode.Encode(torrent)

	err := os.WriteFile("test.torrent", data, 0644)

	if err != nil {
		panic(err)
	}

	fmt.Println("Archivo test.torrent creado con exito")

	// Optionally write the content to a reference file
	_ = os.WriteFile("contenido_ref.bin", content, 0644)

	file, err := os.Open("test.torrent")

	if err != nil {
		panic(err)
	}

	defer file.Close()

	torrentData, err := bencode.Decode(file)

	if err != nil {
		panic(err)
	}

	fmt.Println("\nContenido decodificado")

	for key, value := range torrentData {
		fmt.Printf("%s: %#v\n", key, value)
	}
}
