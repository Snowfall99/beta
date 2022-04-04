package themix

import (
	"log"

	"google.golang.org/protobuf/proto"
	"themix.new.io/message/messagepb"
)

const maxround = 30

func serialCollection(collection *messagepb.Collection) []byte {
	data, err := proto.Marshal(collection)
	if err != nil {
		log.Fatal("proto.Marshal: ", err)
	}
	return data
}

func deserialCollection(data []byte) *messagepb.Collection {
	collection := &messagepb.Collection{}
	err := proto.Unmarshal(data, collection)
	if err != nil {
		log.Fatal("proto.Unmarshal: ", err)
	}
	return collection
}
