package noise

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"

	"github.com/perlin-network/noise"
	"google.golang.org/protobuf/proto"
	"themix.new.io/client/clientpb"
	"themix.new.io/crypto/sha256"
	"themix.new.io/crypto/themixECDSA"
	"themix.new.io/message/messagepb"
)

const BUFFER = 1024

type Peer struct {
	PeerID uint32
	Addr   string
	Pub    *ecdsa.PublicKey
	Ck     *ecdsa.PublicKey
}

type noiseNode struct {
	id           uint32
	peers        map[uint32]*Peer
	node         *noise.Node
	inputc       chan *messagepb.Msg
	outputc      chan *messagepb.Msg
	verifyInput  chan *clientpb.Payload
	verifyOutput chan []byte
	priv         *ecdsa.PrivateKey
	ck           *ecdsa.PublicKey
	sign         bool
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

func InitNoise(id uint32, pk *ecdsa.PrivateKey, ck *ecdsa.PublicKey, peers map[uint32]*Peer, inputc, outputc chan *messagepb.Msg, sign bool) {
	node := &noiseNode{
		id:           id,
		priv:         pk,
		ck:           ck,
		peers:        peers,
		inputc:       inputc,
		outputc:      outputc,
		verifyInput:  make(chan *clientpb.Payload, BUFFER),
		verifyOutput: make(chan []byte, BUFFER),
		sign:         sign}
	go node.initVerifyPool()
	log.Println("noiseNode addr:", peers[id].Addr)
	port, err := strconv.Atoi(strings.Split(peers[id].Addr, ":")[1])
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

func (node *noiseNode) initVerifyPool() {
	for i := 0; i < 2*runtime.NumCPU()-1; i++ {
		go func() {
			for {
				payload := <-node.verifyInput
				if node.verifyPayload(payload) {
					node.verifyOutput <- []byte{1}
				} else {
					node.verifyOutput <- []byte{0}
				}
			}
		}()
	}
	for {
		payload := <-node.verifyInput
		if node.verifyPayload(payload) {
			node.verifyOutput <- []byte{1}
		} else {
			node.verifyOutput <- []byte{0}
		}
	}
}

func (node *noiseNode) verifyReq(req []byte) bool {
	request := &clientpb.Request{}
	err := proto.Unmarshal(req, request)
	if err != nil {
		log.Fatal("proto.Unmarshal: ", err)
	}
	for _, payload := range request.Payload {
		node.verifyInput <- payload
	}
	result := true
	for i := 0; i < len(request.Payload); i++ {
		resp := <-node.verifyOutput
		if resp[0] == 0 {
			log.Println("verify request fail")
			result = false
		}
	}
	if !result {
		return false
	}
	return true
}

func (node *noiseNode) verifyPayload(payload *clientpb.Payload) bool {
	content := []byte(payload.Payload)
	hash, err := sha256.ComputeHash(content)
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
	}
	b, err := themixECDSA.VerifyECDSA(node.ck, payload.Signature, hash)
	if err != nil {
		log.Fatal("themix.VerifyECDSA: ", err)
	}
	return b
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
	if msg.Type == messagepb.MsgType_VAL || msg.Type == messagepb.MsgType_ECHO ||
		msg.Type == messagepb.MsgType_BVAL || msg.Type == messagepb.MsgType_AUX {
		if !verify(msg, node.peers[msg.From].Pub) {
			log.Fatal("verify: consensus message verification fail")
		}
		if node.sign && msg.Type == messagepb.MsgType_VAL {
			if !node.verifyReq(msg.Content) {
				log.Fatal("verifyReq: client request payload verification fail")
			}
		}
	}
	node.inputc <- msg
}

func (node *noiseNode) broadcast() {
	for {
		msg := <-node.outputc
		msg.From = node.id
		sign(msg, node.priv)
		for _, peer := range node.peers {
			if peer != nil {
				go node.sendMessage(peer.Addr, msg)
			}
		}
	}
}

func (node *noiseNode) sendMessage(addr string, msg *messagepb.Msg) {
	m := noiseMessage{Msg: msg}
	err := node.node.SendMessage(context.TODO(), addr, m)
	if err != nil {
		log.Println("node.node.SendMessage: ", err)
	}
}

func verify(msg *messagepb.Msg, pub *ecdsa.PublicKey) bool {
	content := getMsgInfo(msg)
	hash, err := sha256.ComputeHash(content)
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
	}
	b, err := themixECDSA.VerifyECDSA(pub, msg.Signature, hash)
	if err != nil {
		log.Fatal("themixECDSA.VerifyECDSA: ", err)
	}
	return b
}

func sign(msg *messagepb.Msg, priv *ecdsa.PrivateKey) {
	content := getMsgInfo(msg)
	hash, err := sha256.ComputeHash(content)
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
	}
	sig, err := themixECDSA.SignECDSA(priv, hash)
	if err != nil {
		log.Fatal("themixECDSA.SignECDSA: ", err)
	}
	msg.Signature = sig
}

func getMsgInfo(msg *messagepb.Msg) []byte {
	btype := make([]byte, 8)
	binary.LittleEndian.PutUint64(btype, uint64(msg.Type))
	bseq := make([]byte, 8)
	binary.LittleEndian.PutUint64(bseq, uint64(msg.Seq))
	bproposer := make([]byte, 8)
	binary.LittleEndian.PutUint64(bproposer, uint64(msg.Proposer))
	b := make([]byte, 26)
	b = append(b, btype...)
	b = append(b, bseq...)
	b = append(b, bproposer...)
	b = append(b, uint8(msg.Round))
	if len(msg.Content) > 0 {
		b = append(b, uint8(msg.Content[0]))
	} else {
		b = append(b, uint8(0))
	}
	return b
}
