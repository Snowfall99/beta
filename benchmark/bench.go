package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/client/clientpb"
	"themix.new.io/config/configpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/crypto/themixECDSA"
	"themix.new.io/noise"
	"themix.new.io/themix"
)

const INTERVAL = 5
const IPADDR = "localhost"

func initLog() {
	file, err := os.OpenFile("bench.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	log.SetOutput(file)
}

func main() {
	data, err := os.ReadFile("bench.config")
	if err != nil {
		panic(err)
	}
	var configuration configpb.Configuration
	err = prototext.Unmarshal(data, &configuration)
	if err != nil {
		panic(err)
	}
	initLog()
	for i := 0; i < int(configuration.N); i++ {
		blsSig, err := bls.InitBLS(configuration.BlsKeyPath,
			len(configuration.Peers), int(len(configuration.Peers)/2+1), i)
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
		for j := 0; j < int(configuration.N); j++ {
			clients = append(clients, configuration.Peers[j].Client)
		}
		node := themix.InitNode(uint32(i), blsSig, int(configuration.Batch),
			int(configuration.N), int(configuration.F),
			int(configuration.Delta), int(configuration.DeltaBar),
			pk, &ck.PublicKey, peers, clients, configuration.Sign)
		go node.Run()
	}
	txCh := make(chan []byte, configuration.N)
	for i := 0; i < int(configuration.N); i++ {
		go func(id int, port string) {
			conn, err := grpc.Dial(IPADDR+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatal("grpc.Dial: ", err)
			}
			defer conn.Close()

			client := clientpb.NewThemixClient(conn)
			for {
				_, err := client.Post(context.TODO(), &clientpb.Request{Payload: []*clientpb.Payload{{Payload: "a"}}})
				if err != nil {
					log.Fatal("client.Post: ", err)
				}
				txCh <- []byte{}
			}
		}(i, configuration.Peers[i].Client)
	}
	tx := 0
	totalTime := 0
	ticker := time.NewTicker(INTERVAL * time.Second)
	for {
		select {
		case <-txCh:
			tx++
		case <-ticker.C:
			totalTime += INTERVAL
			fmt.Printf("tps: %.2f tx/s\n", float64(tx)/float64(totalTime))
		}
	}
}
