package euphoria

import (
	"encoding/json"
	"errors"
)

func Encode(kind string, v any) (b []byte, err error) {
	switch kind {
	case "json":
		return json.Marshal(v)
	default:
		return nil, errors.New("unknown encoding: " + kind)
	}
}

func Decode(kind string, b []byte, v any) (err error) {
	switch kind {
	case "json":
		return json.Unmarshal(b, v)
	default:
		return errors.New("unknown encoding: " + kind)
	}
}
