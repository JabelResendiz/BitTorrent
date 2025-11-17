package client

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"src/bencode"
	"strings"
)

type ClientConfig struct {
	TorrentPath     string
	ArchivesDir     string
	Hostname        string
	PeerId          string
	InfoHash        [20]byte
	InfoHashEncoded string
	AnnounceURL     string
	FileLength      int64
	PieceLength     int64
	ExpectedHashes  [][20]byte
	FileName        string
}

func ParseFlags() (string, string, string, string, string, int) {
	torrentFlag := flag.String("torrent", "", "ruta al archivo .torrent (obligatorio)")
	archivesFlag := flag.String("archives", "./archives", "directorio de archivos donde guardar/leer archivos")
	hostnameFlag := flag.String("hostname", "", "nombre de host para announces (requerido en Docker/NAT)")
	discoveryFlag := flag.String("discovery-mode", "tracker", "discovery mode: tracker|overlay")
	bootstrapFlag := flag.String("bootstrap", "", "comma-separated bootstrap peers para overlay (host:port)")
	overlayPortFlag := flag.Int("overlay-port", 6000, "puerto donde escucha el overlay (TCP)")

	flag.Parse()

	if *torrentFlag == "" {
		fmt.Println("Error: debe especificar --torrent=/ruta/al/archivo.torrent")
		os.Exit(2)
	}

	return *torrentFlag, *archivesFlag, *hostnameFlag, *discoveryFlag, *bootstrapFlag, *overlayPortFlag
}

func LoadTorrentMetadata(torrentPath, archivesPath string) *ClientConfig {
	archivesDir := archivesPath
	if strings.HasPrefix(archivesDir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if archivesDir == "~" {
				archivesDir = home
			} else if strings.HasPrefix(archivesDir, "~/") {
				archivesDir = filepath.Join(home, archivesDir[2:])
			}
		}
	}

	if err := os.MkdirAll(archivesDir, 0755); err != nil {
		fmt.Println("No se pudo crear directorio:", err)
		os.Exit(1)
	}

	// Abrir y decodificar el .torrent
	torrent, err := os.Open(torrentPath)
	if err != nil {
		panic(err)
	}
	defer torrent.Close()

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
	var pieceLength int64
	if v, ok := info["piece length"].(int64); ok {
		pieceLength = v
	}

	var expectedHashes [][20]byte
	if piecesRaw, ok := info["pieces"].(string); ok {
		numPieces := int((length + pieceLength - 1) / pieceLength)
		if len(piecesRaw) == numPieces*20 {
			expectedHashes = make([][20]byte, numPieces)
			for i := 0; i < numPieces; i++ {
				copy(expectedHashes[i][:], piecesRaw[i*20:(i+1)*20])
			}
		} else {
			fmt.Printf("Advertencia: longitud de 'piece' (%d) no coincide con numPiece*20 (%d)\n", len(piecesRaw), numPieces*20)
		}
	}

	var buf strings.Builder
	for _, b := range infoHash {
		buf.WriteString(fmt.Sprintf("%%%02X", b))
	}

	outName := "archivo.bin"
	if n, ok := info["name"].(string); ok && n != "" {
		outName = filepath.Base(n)
	}

	cfg := &ClientConfig{
		TorrentPath:     torrentPath,
		ArchivesDir:     archivesDir,
		PeerId:          GeneratePeerId(),
		InfoHash:        infoHash,
		InfoHashEncoded: buf.String(),
		AnnounceURL:     announce,
		FileLength:      length,
		PieceLength:     pieceLength,
		ExpectedHashes:  expectedHashes,
		FileName:        outName,
	}

	return cfg
}

func (cfg *ClientConfig) GetStoragePaths() (tempPath, finalPath string) {
	tempPath = filepath.Join(cfg.ArchivesDir, cfg.FileName+".part")
	finalPath = filepath.Join(cfg.ArchivesDir, cfg.FileName)
	return
}
