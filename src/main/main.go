


package main

import (
	"os"
	"fmt"
	"src/bencode"
)


func main(){
	torrent := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info": map[string]interface{}{
			"name":         "archivo.txt",
			"length":       int64(12345),
			"piece length": int64(16384),
			"pieces":       "12345678901234567890", // dummy SHA1
		},
	}

	data := bencode.Encode(torrent)

	err := os.WriteFile("test.torrent", data,0644)

	if err != nil {
		panic(err)
	}

	fmt.Println("Archivo test.torrent creado con exito")



	file, err := os.Open("test.torrent")

	if err != nil {
		panic(err)
	}


	defer file.Close()


	torrentData,err  := bencode.Decode(file)

	if err != nil {
		panic(err)
	}

	fmt.Println("\nContenido decodificado")

	for key, value := range torrentData {
		fmt.Printf("%s: %#v\n", key, value)
	}
}