package euphoria

import (
	"fmt"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	var err error
	kind := "application/msgpack"
	e1 := &Event{
		Nm: "e1",
		To: "foo",
		Fm: "bar",
		Tm: time.Now().UnixNano(),
		Dt: []byte{1, 2, 3},
	}
	e2 := &Event{
		Nm: "e2",
		To: "foo",
		Fm: "bar",
		Tm: time.Now().UnixNano(),
		Dt: []byte{4, 5, 6},
	}
	// test encode
	ees := []*Event{e1, e2}
	b, err := Encode(kind, &ees)
	if err != nil {
		panic(ees)
	}
	fmt.Println("encode result", len(b), "bytes")
	// test decode
	var eds []*Event
	err = Decode(kind, b, &eds)
	if err != nil {
		panic(err)
	}
	fmt.Println("decode result:", eds[0], eds[1])
}
