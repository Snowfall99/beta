package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"themix.new.io/client/clientpb"
	"themix.new.io/crypto/sha256"
	"themix.new.io/crypto/themixECDSA"
)

var (
	addr    = flag.String("addr", "localhost:11200", "address of the themix Node")
	ckpath  = flag.String("ck", "../crypto/", "client ecdsa key path")
	number  = flag.Int("num", 1, "number of requests")
	batch   = flag.Int("batch", 1, "batch size")
	payload = flag.Int("payload", 600, "payload size")
)

func main() {
	flag.Parse()
	file, err := os.OpenFile("node.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal("os.OpenFile: ", err)
	}
	defer file.Close()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("grpc.Dial: ", err)
	}
	defer conn.Close()

	client := clientpb.NewThemixClient(conn)
	// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	// defer cancel()

	ck, err := themixECDSA.LoadKey(*ckpath)
	if err != nil {
		log.Fatal("themixECDSA.LoadKey: ", err)
	}

	start := time.Now()
	for i := 0; i < *number; i++ {
		request := genClientRequest(*batch, *payload, ck)
		_, err := client.Post(context.Background(), request)
		if err != nil {
			log.Fatal("client.Post: ", err)
		}
		file.Write([]byte("ok"))
	}
	end := time.Now()
	file.Write([]byte(fmt.Sprintf("finish %d txs in %d", *number**batch, end.Sub(start).Milliseconds())))
}

func genClientRequest(batchsize, payloadsize int, ck *ecdsa.PrivateKey) *clientpb.Request {
	payload := &clientpb.Payload{}
	for i := 0; i < payloadsize; i++ {
		payload.Payload += "a"
	}
	hash, err := sha256.ComputeHash([]byte(payload.Payload))
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
		return nil
	}
	sign, err := themixECDSA.SignECDSA(ck, hash)
	if err != nil {
		log.Fatal("themixECDSA.SignECDSA: ", err)
	}
	payload.Signature = sign
	request := &clientpb.Request{
		Payload: make([]*clientpb.Payload, batchsize),
	}
	for i := 0; i < batchsize; i++ {
		request.Payload[i] = payload
	}
	return request
}
