package main

import (
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"themix.new.io/config/configpb"
	"themix.new.io/themix"
)

func main() {
	// TODO(chenzx): To be implemented.
	data, err := os.ReadFile("themix.config")
	if err != nil {
		panic(err)
	}

	var configuration configpb.Configuration
	err = prototext.Unmarshal(data, &configuration)
	if err != nil {
		panic(err)
	}
	node := themix.InitNode(configuration.Id, configuration.Batch, configuration.Peers)
	node.Run()
}
