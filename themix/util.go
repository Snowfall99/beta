package themix

import (
	"crypto/ecdsa"
	"log"

	"google.golang.org/protobuf/proto"
	"themix.new.io/crypto/sha256"
	"themix.new.io/crypto/themixECDSA"
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

func verify(content, sign []byte, pub *ecdsa.PublicKey) bool {
	hash, err := sha256.ComputeHash(content)
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
	}
	b, err := themixECDSA.VerifyECDSA(pub, sign, hash)
	if err != nil {
		log.Fatal("themixECDSA.VerifyECDSA: ", err)
	}
	return b
}
