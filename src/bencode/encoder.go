package bencode

import (
	"bytes"
	"reflect"
	"sort"
	"strconv"
)



type encoder struct {
	bytes.Buffer
}


func (encoder * encoder) encoderInt(v int64){
	encoder.WriteByte('i')
	defer encoder.WriteByte('e')
	encoder.WriteString(strconv.FormatInt(v,10))
	
}


func (encoder * encoder) encoderUInt(v uint64){
	encoder.WriteByte('i')
	defer encoder.WriteByte('e')
	encoder.WriteString(strconv.FormatUint(v,10))
	
}

func (encoder * encoder) encoderString(v string){
	encoder.WriteString(strconv.Itoa(len(v)))
	encoder.WriteByte(':')
	encoder.WriteString(v)
}

func (encoder * encoder)encoderList(v [] interface{}){
	encoder.WriteByte('l')

	defer encoder.WriteByte('e')

	for _,value := range v {
		encoder.encoderInterfaceType(value)
	}

}


func (encoder * encoder) encoderDictionary(v  map[string]interface{}) {

	list := make(sort.StringSlice, len(dict))

	i:=0

	for key := range v {
		list[i++] = key
	}

	list.Sort()

	encoder.WriteByte('d')
	defer encoder.WriteByte('e')

	for _,key := range list {
		encoder.encoderString(key)
		encoder.encoderInterfaceType(v[key])
	}
}



func (encoder * encoder) encoderInterfaceType( v interface{}){
	switch v:= v.(type){
	case int, int8,int16,int32,int64:
		encoder.encoderInt(reflect.ValueOf(v).Int())
	case uint,uint8,uint16,uint32,uint64:
		encoder.encoderUInt(reflect.ValueOf(v).Uint())
	case string:
		encoder.encoderString(v)
	case []interface{}:
		encoder.encoderList(v)
	case map[string]interface{}:
		encoder.encoderDictionary(v)
	default:
		panic("Encoder not valid")
	}
}



func Encode(v interface{}) []byte{
	encoder := encoder{}
	encoder.encoderInterfaceType(v)
	return encoder.Bytes()
}