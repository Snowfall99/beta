package noise

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/perlin-network/noise"
	"google.golang.org/protobuf/proto"
	"themix.new.io/message/messagepb"
)

type peer struct {
	peerID uint32
	addr   string
}

type noiseNode struct {
	id      uint32
	peers   map[uint32]*peer
	node    *noise.Node
	inputc  chan *messagepb.Msg
	outputc chan *messagepb.Msg
}

type noiseMessage struct {
	Msg *messagepb.Msg
}

func (msg noiseMessage) Marshal() []byte {
	data, err := proto.Marshal(msg.Msg)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func unmarshalNoiseMessage(buf []byte) (noiseMessage, error) {
	msg := noiseMessage{Msg: new(messagepb.Msg)}
	err := proto.Unmarshal(buf, msg.Msg)
	if err != nil {
		return noiseMessage{}, err
	}
	return msg, nil
}

func InitNoise(id uint32, peersInfo map[uint32]string, inputc, outputc chan *messagepb.Msg) {
	peers := make(map[uint32]*peer)
	for idx, peerInfo := range peersInfo {
		peers[idx] = &peer{idx, peerInfo}
	}
	node := &noiseNode{
		id:      id,
		peers:   peers,
		inputc:  inputc,
		outputc: outputc}
	log.Println("noiseNode addr:", peersInfo[id])
	port, err := strconv.Atoi(strings.Split(peersInfo[id], ":")[1])
	if err != nil {
		log.Fatal("strconv.Atoi: ", err)
	}
	n, err := noise.NewNode(noise.WithNodeBindHost(net.ParseIP("127.0.0.1")),
		noise.WithNodeBindPort(uint16(port)))
	if err != nil {
		log.Fatal("noise.NewNode: ", err)
	}
	node.node = n
	node.node.RegisterMessage(noiseMessage{}, unmarshalNoiseMessage)
	node.node.Handle(node.Handler)
	err = node.node.Listen()
	if err != nil {
		log.Fatal("node.node.Listen: ", err)
	}
	go node.broadcast()
}

func (node *noiseNode) Handler(ctx noise.HandlerContext) error {
	obj, err := ctx.DecodeMessage()
	if err != nil {
		log.Fatal("ctx.DecodeMessage: ", err)
	}
	msg, ok := obj.(noiseMessage)
	if !ok {
		log.Fatal("obj.(noiseMessage): ", err)
	}
	go node.onReceiveMessage(msg.Msg)
	return nil
}

func (node *noiseNode) onReceiveMessage(msg *messagepb.Msg) {
	node.inputc <- msg
}

func (node *noiseNode) broadcast() {
	for {
		msg := <-node.outputc
		for _, peer := range node.peers {
			if peer != nil {
				go node.sendMessage(peer.addr, msg)
			}
		}
	}
}

func (node *noiseNode) sendMessage(addr string, msg *messagepb.Msg) {
	m := noiseMessage{Msg: msg}
	err := node.node.SendMessage(context.TODO(), addr, m)
	if err != nil {
		log.Fatal("node.node.SendMessage: ", err)
	}
}
