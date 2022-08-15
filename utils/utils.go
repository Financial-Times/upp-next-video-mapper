package utils

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
)

func IsValidXHTML(data string) bool {
	d := xml.NewDecoder(strings.NewReader(data))

	valid := true
	for {
		_, err := d.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			valid = false
			break
		}
	}

	return valid
}

func UnsafeJSONMarshal(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)

	return b, nil
}

func GetPrefixedURL(prefix string, uuid string) string {
	return prefix + uuid
}
