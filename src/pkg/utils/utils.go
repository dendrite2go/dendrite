package utils

import (
	"log"

	"github.com/golang/protobuf/proto"
)

func ReportError(label string, f func() error) {
	e := f()
	if e != nil {
		log.Printf("%v: Error: %v", label, e)
	}
}

func ProtoCast(source proto.Message, destination proto.Message) error {
	bytes, e := proto.Marshal(source)
	if e != nil {
		return e
	}
	return proto.Unmarshal(bytes, destination)
}
