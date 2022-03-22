package main

import (
	"context"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"themix.new.io/client/clientpb"
)

var (
	addr   = flag.String("addr", "localhost:11200", "address of the themix Node")
	number = flag.Int("num", 1, "number of requests")
)

func main() {
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("grpc.Dial: ", err)
	}
	defer conn.Close()

	client := clientpb.NewThemixClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for i := 0; i < *number; i++ {
		resp, err := client.Post(ctx, &clientpb.Request{})
		if err != nil {
			log.Fatal("client.Post: ", err)
		}
		log.Println("resp: ", resp.GetOk())
	}
}
