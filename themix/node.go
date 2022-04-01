package themix

import (
	"crypto/ecdsa"

	"themix.new.io/client/clientpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
	"themix.new.io/noise"
)

type Node struct {
	// proposer  *Proposer
	themixQue *ThemixQue
}

func InitNode(id uint32, blsSig *bls.BlsSig, batch, n, f, delta, deltaBar int, pk *ecdsa.PrivateKey, peers map[uint32]*noise.Peer, clients []string) *Node {
	inputc := make(chan *messagepb.Msg)
	outputc := make(chan *messagepb.Msg)
	reqc := make(chan *clientpb.Request)
	repc := make(chan []byte)
	client := clients[id]
	go initProposer(batch, client, reqc, repc, outputc, id)
	themixQue := initThemixQue(id, blsSig, n, f, delta, deltaBar, inputc, outputc, reqc, repc)
	noise.InitNoise(id, pk, peers, inputc, outputc)
	return &Node{
		themixQue: themixQue,
	}
}

func (node *Node) Run() {
	// go node.proposer.run()
	node.themixQue.run()
}
