package main

import (
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/config/configpb"
	"themix.new.io/crypto/themixBLS"
	"themix.new.io/crypto/themixECDSA"
	"themix.new.io/noise"
	"themix.new.io/themix"
)

func initLog() {
	err := os.Mkdir("log", 0600)
	if err != nil && !os.IsExist(err) {
		panic(fmt.Sprint("os.Mkdir: ", err))
	}
	file, err := os.OpenFile("./log/node.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		panic(fmt.Sprint("os.OpenFile: ", err))
	}
	log.SetOutput(file)
}

func main() {
	// id is set as a flag parameter for local test
	// id := flag.Uint("id", 0, "ID of themix node")
	// flag.Parse()
	initLog()
	data, err := os.ReadFile("themix.config")
	if err != nil {
		panic(fmt.Sprint("os.ReadFile: ", err))
	}
	var configuration configpb.Configuration
	err = prototext.Unmarshal(data, &configuration)
	if err != nil {
		panic(fmt.Sprint("protext.Unmarshal: ", err))
	}
	blsSig, err := themixBLS.InitBLS(configuration.BlsKeyPath,
		len(configuration.Peers), int(len(configuration.Peers)/2+1), int(configuration.Id))
	if err != nil {
		panic(fmt.Sprint("themixBLS.InitBLS: ", err))
	}
	pk, err := themixECDSA.LoadKey(configuration.Pk)
	if err != nil {
		panic(fmt.Sprint("themixECDSA.LoadKey: ", err))
	}
	ck, err := themixECDSA.LoadKey(configuration.Ck)
	if err != nil {
		panic(fmt.Sprint("themixECDSA.LoadKey: ", err))
	}
	peers := make(map[uint32]*noise.Peer)
	for idx, peerInfo := range configuration.Peers {
		peers[uint32(idx)] = &noise.Peer{
			PeerID: peerInfo.Id,
			Addr:   peerInfo.Addr,
			Pub:    &pk.PublicKey,
			Ck:     &ck.PublicKey,
		}
	}
	var clients []string
	for i := 0; i < int(configuration.N); i++ {
		clients = append(clients, configuration.Peers[i].Client)
	}
	node := themix.InitNode(uint32(configuration.Id), blsSig, int(configuration.Batch),
		int(configuration.N), int(configuration.F),
		int(configuration.Delta), int(configuration.DeltaBar),
		pk, &ck.PublicKey, peers, clients, configuration.Sign)
	node.Run()
}
