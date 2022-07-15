package themix

import (
	"crypto/ecdsa"
	"log"

	"themix.new.io/client/clientpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
	"themix.new.io/noise"
)

type Node struct {
	// proposer  *Proposer
	themixQue *ThemixQue
}

func InitNode(id uint32, blsSig *bls.BlsSig, batch, n, f, delta, deltaBar int, pk *ecdsa.PrivateKey, ck *ecdsa.PublicKey, peers map[uint32]*noise.Peer, clients []string, sign bool) *Node {
	inputc := make(chan *messagepb.Msg)
	outputc := make(chan *messagepb.Msg)
	reqc := make(chan *clientpb.Request)
	repc := make(chan []byte)
	client := clients[id]
	go initProposer(batch, client, reqc, repc, outputc, id)
	pubkeys := make(map[uint32]*ecdsa.PublicKey)
	for _, peer := range peers {
		pubkeys[peer.PeerID] = peer.Pub
	}
	themixQue := initThemixQue(id, blsSig, n, f, delta, deltaBar, inputc, outputc, reqc, repc, pubkeys)
	noise.InitNoise(id, pk, ck, peers, inputc, outputc, sign)
	return &Node{
		themixQue: themixQue,
	}
}

func (node *Node) Run() {
	// go node.proposer.run()
	log.Print("Node is running")
	node.themixQue.run()
}
