package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"os"
)

func decodeBencode(data *bytes.Buffer) (interface{}, error) {
	b, err := data.ReadByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case 'i':
		num, err := readInt(data)
		if err != nil {
			return nil, err
		}
		return num, nil

	case 'l': 
		var list []interface{}
		for {
			if peek, _ := data.ReadByte(); peek == 'e' {
				break
			} else {
				data.UnreadByte()
			}
			val, err := decodeBencode(data)
			if err != nil {
				return nil, err
			}
			list = append(list, val)
		}
		return list, nil

	case 'd': 
		dict := make(map[string]interface{})
		for {
			if peek, _ := data.ReadByte(); peek == 'e' {
				break
			} else {
				data.UnreadByte()
			}

			keyRaw, err := decodeBencode(data)
			if err != nil {
				return nil, err
			}
			key, ok := keyRaw.(string)
			if !ok {
				return nil, errors.New("dictionary key must be string")
			}

			val, err := decodeBencode(data)
			if err != nil {
				return nil, err
			}

			dict[key] = val
		}
		return dict, nil

	default: 
		if b < '0' || b > '9' {
			return nil, errors.New("invalid bencode format")
		}
		data.UnreadByte()
		str, err := readString(data)
		if err != nil {
			return nil, err
		}
		return str, nil
	}
}


func readInt(data *bytes.Buffer) (int, error) {
	numBytes, err := data.ReadBytes('e')
	if err != nil {
		return 0, err
	}
	numStr := string(numBytes[:len(numBytes)-1]) // quitar 'e'
	return strconv.Atoi(numStr)
}

func readString(data *bytes.Buffer) (string, error) {
	lenBytes, err := data.ReadBytes(':')
	if err != nil {
		return "", err
	}
	length, err := strconv.Atoi(string(lenBytes[:len(lenBytes)-1]))
	if err != nil {
		return "", err
	}

	strBytes := make([]byte, length)
	_, err = data.Read(strBytes)
	if err != nil {
		return "", err
	}
	return string(strBytes), nil
}

func main() {
	filename := "ejemplo.torrent"

	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	val, err := decodeBencode(bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	torrent, ok := val.(map[string]interface{})
	if !ok {
		panic("El archivo torrent no es un diccionario en la raíz")
	}

	for k, v := range torrent {
		fmt.Printf("%s: %T -> %v\n", k, v, v)
	}

	if info, ok := torrent["info"].(map[string]interface{}); ok {
		fmt.Println("\n--- Info ---")
		for k, v := range info {
			fmt.Printf("%s: %T -> %v\n", k, v, v)
		}
	}
}
