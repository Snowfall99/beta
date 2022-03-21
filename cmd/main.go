package main

import (
	"flag"
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/config/configpb"
	"themix.new.io/themix"
)

func main() {
	// TODO(chenzx): Under construction.
	// id is set as a flag parameter for local test
	id := flag.Uint("id", 0, "ID of themix node")
	flag.Parse()
	data, err := os.ReadFile("themix.config")
	if err != nil {
		panic(err)
	}
	var configuration configpb.Configuration
	err = prototext.Unmarshal(data, &configuration)
	if err != nil {
		panic(err)
	}
	node := themix.InitNode(uint32(*id), configuration.Batch, configuration.N, configuration.Peers)
	node.Run()
}
