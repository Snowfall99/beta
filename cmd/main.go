package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/config/configpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/crypto/themixECDSA"
	"themix.new.io/noise"
	"themix.new.io/themix"
)

func initLog(id int) {
	file, err := os.OpenFile(fmt.Sprintf("../log/%d.log", id), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	log.SetOutput(file)
}

func main() {
	// id is set as a flag parameter for local test
	id := flag.Uint("id", 0, "ID of themix node")
	flag.Parse()
	initLog(int(*id))
	data, err := os.ReadFile("themix.config")
	if err != nil {
		panic(err)
	}
	var configuration configpb.Configuration
	err = prototext.Unmarshal(data, &configuration)
	if err != nil {
		panic(err)
	}
	blsSig, err := bls.InitBLS(configuration.BlsKeyPath,
		len(configuration.Peers), int(len(configuration.Peers)/2+1), int(*id))
	if err != nil {
		panic(err)
	}
	pk, err := themixECDSA.LoadKey(configuration.Pk)
	if err != nil {
		panic(err)
	}
	ck, err := themixECDSA.LoadKey(configuration.Ck)
	if err != nil {
		panic(err)
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
	node := themix.InitNode(uint32(*id), blsSig, int(configuration.Batch),
		int(configuration.N), int(configuration.F),
		int(configuration.Delta), int(configuration.DeltaBar),
		pk, &ck.PublicKey, peers, clients, configuration.Sign)
	node.Run()
}
