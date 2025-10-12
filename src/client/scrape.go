package client
//client/scraper.go

import (
	"fmt"
	"src/bencode"
	"crypto/sha1"
	"net/http"
	"strings"
	"os"
)
// implementacion de un scraper

// leer el .torrent (par extraer el info_hash)
// construir la URL de scrape, si es valida
// añadir el info_hash con parametros
// hacer el http.Get() y decodificar el diccionario bencode



func main(){

	torrent, err := os.Open("test.torrent")

	if err != nil {
		panic(err)
	}

	meta,err := bencode.Decode(torrent)

	if err != nil{
		panic(err)
	}

	announce := meta["announce"].(string)
	info := meta["info"].(map[string]interface{})



	// vamos a construir la URL de scrape, si es valida
	pos := strings.LastIndex(announce,"/")

	if pos == -1 {
		fmt.Println("URL invalida , debe contener '/'")
		return
	}

	last := announce[pos+1:]

	if !strings.HasPrefix(last,"announce"){
		fmt.Println("tracker no contiene capacidad para scrapear")
		return 
	}

	newLast := strings.Replace(last,"announce","scrape",1)
	
	newURL := announce[:pos+1]+ newLast

	// añadir el info hash
	infoEncoded := bencode.Encode(info)
	infoHash := sha1.Sum(infoEncoded)


	var buf strings.Builder
	for _, b := range infoHash {
		buf.WriteString(fmt.Sprintf("%%%02X", b))
	}

	/// hacer el http.get() y decodificar el diccionario bencode
	fullURL := newURL + "?info_hash=" + buf.String()

	fmt.Println("Tracker request: ",fullURL)

	resp, err := http.Get(fullURL)

	if err != nil{
		panic(err)
	}

	resp.Body.Close()
	
}