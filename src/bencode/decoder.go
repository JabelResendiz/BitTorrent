
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

// Read a number from the decoder (stream) until a given byte (like ':' or 'e')
func (decoder * decoder) decoderIntU(until byte) (interface {}, error){
	res, err := decoder.ReadSlice(until)

	if err != nil {
		return nil, err
	}

	str := string(res[:len(res)-1])


	if value, err := strconv.ParseInt(str, 10,64) ; err == nil {
		return value,nil
	} else if value, err := strconv.ParseUint(str,10,64) ; err == nil {
		return value, nil
	}

	return nil, err
}

func (decoder * decoder) decoderInt() (interface{}, error){
	return decoder.decoderIntU('e')
}

func (decoder * decoder) decoderList() ([]interface{}, error){
	var list []interface{}

	for {
		ch, err := decoder.ReadByte()

		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			break
		}

		item , err := decoder.decoderInterfaceType(ch)

		if err != nil {
			return nil, err
		}

		list = append(list,item)
	}

	return list, nil
}

func (decoder * decoder) decoderInterfaceType(id byte) (interface{}, error) {

	switch id {
	case 'i' :
		return decoder.decoderInt()
	case 'l':
		return decoder.decoderList()
	case 'd':
		return decoder.decoderDictionary()
	default:
		if err := decoder.UnreadByte() ; err != nil {
			return nil, err
		}

		return decoder.decoderString()
	}

	// return nil, nil
}

func (decoder * decoder) decoderString() ( string, error){

	len, err := decoder.decoderIntU(':')

	if err != nil {
		return "", err
	}

	var stringLength int64
	var ok bool

	if stringLength, ok = len.(int64) ; !ok {
		return "", errors.New("string length maay not exceed the size of int64")
	}

	if stringLength < 0 {
		return "", errors.New("string length can not be a negative number")
	}

	buffer := make([]byte, stringLength)
	_, err = io.ReadFull(decoder,buffer)

	return string(buffer), err
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

		dict[key] = interf

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
		return nil , errors.New("bencode data must begin with a dictionary")
	}

	return decoder.decoderDictionary()
}