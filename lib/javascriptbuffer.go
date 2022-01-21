package lib

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type JavaScriptBuffer struct {
	Data []uint8
}

func (buf *JavaScriptBuffer) MarshalJSON() ([]byte, error) {
	var array string
	if buf.Data == nil {
		array = "null"
	} else {
		array = strings.Join(strings.Fields(fmt.Sprintf("%d", buf.Data)), ",")
	}
	jsonResult := fmt.Sprintf(`{"type": "Buffer", "data":%s}`, array)
	return []byte(jsonResult), nil
}

func ToJavaScriptBuffer(hexString string) (*JavaScriptBuffer, error) {
	buf := JavaScriptBuffer{}
	hexArray, err := hex.DecodeString(hexString)
	if err != nil {
		return &buf, err
	}
	buf.Data = hexArray
	return &buf, nil
}
