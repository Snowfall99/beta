package themix

import (
	"themix.new.io/client/clientpb"
	"themix.new.io/common/messagepb"
	"themix.new.io/config/configpb"
	"themix.new.io/noise"
)

type Node struct {
	// proposer  *Proposer
	themixQue *ThemixQue
}

func InitNode(id, batch uint32, peers []*configpb.Peer) *Node {
	inputc := make(chan *messagepb.Msg)
	outputc := make(chan *messagepb.Msg)
	reqc := make(chan *clientpb.Request)
	repc := make(chan []byte)
	client := peers[id].Client
	go initProposer(batch, client, reqc, repc, outputc)
	themixQue := initThemixQue(inputc, outputc, repc)
	peerInfo := make(map[uint32]string)
	for _, peer := range peers {
		peerInfo[peer.Id] = peer.Addr
	}
	noise.InitNoise(id, peerInfo, inputc, outputc)
	return &Node{
		themixQue: themixQue,
	}
}

func (node *Node) Run() {
	// go node.proposer.run()
	node.themixQue.run()
}
