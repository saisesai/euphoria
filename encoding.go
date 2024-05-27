package euphoria

import (
	"encoding/json"
	"errors"
	"github.com/vmihailenco/msgpack/v5"
)

func Encode(kind string, v any) (b []byte, err error) {
	switch kind {
	case "application/json":
		return json.Marshal(v)
	case "application/msgpack":
		return msgpack.Marshal(v)
	default:
		return nil, errors.New("unknown encoding: " + kind)
	}
}

func Decode(kind string, b []byte, v any) (err error) {
	switch kind {
	case "application/json":
		return json.Unmarshal(b, v)
	case "application/msgpack":
		return msgpack.Unmarshal(b, v)
	default:
		return errors.New("unknown encoding: " + kind)
	}
}
