package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/config/configpb"
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
	// TODO(chenzx): Under construction.
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
	node := themix.InitNode(uint32(*id), int(configuration.Batch), int(configuration.N), int(configuration.F), int(configuration.Delta), int(configuration.DeltaBar), configuration.Peers)
	node.Run()
}
