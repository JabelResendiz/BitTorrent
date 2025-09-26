
package bencode


import (
	"bufio"
	"errors"
	"io"
	"strconv"
)



type decoder struct {
	bufio.Reader
}




func (decoder * decoder) decoderInterfaceType(id byte) (interface{}, error) {

}

func (decoder * decoder) decoderString() ( string, error){

}



func (decoder * decoder) decoderDictionary() (map[string]interface{}, error) {

	dict := make(map[string]interface{})

	for {
		key, err := decoder.decoderString()

		if err != nil {
			return nil, err
		}

		ch, err := decoder.ReadByte()

		if err != nil {
			return nil, err
		}

		interf, err := decoder.decoderInterfaceType(ch)
		if err != nil {
			return nil, err
		}

		dict[key] = item

		nextbyte, err := decoder.ReadByte()
		if err != nil {
			return nil, err
		}

		if nextbyte == 'e' {
			break
		} else if err := decoder.UnreadByte() ; err != nil {
			return nil, err
		}
	}

	return dict, nil
}

func Decode(reader io.Reader) (map[string]interface{}, error){
	decoder := decoder{*bufio.NewReader(reader)}

	if firstbyte, err := decoder.ReadByte(); err != nil {
		return make (map[string]interface{}), nil
	}else if firstbyte != 'd' {
		return nil , error.New("bencode data must begin with a dictionary")
	}

	return decoder.decoderDictionary()
}