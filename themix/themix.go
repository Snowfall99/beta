package themix

import (
	"log"

	"themix.new.io/common/messagepb"
)

type Themix struct {
	// TODO(chenzx): To be implemented.
	inputc  chan *messagepb.Msg
	msgc    map[uint32]chan *messagepb.Msg
	repc    chan []byte
	n       uint32
	decided uint32
}

func initThemix(inputc chan *messagepb.Msg, repc chan []byte, n uint32) *Themix {
	themix := &Themix{
		inputc: inputc,
		repc:   repc,
		msgc:   make(map[uint32]chan *messagepb.Msg),
		n:      n,
	}
	for i := 0; i < int(n); i++ {
		msgc := make(chan *messagepb.Msg)
		themix.msgc[uint32(i)] = msgc
		initInstance(msgc, n)
	}
	return themix
}

func (themix *Themix) run() {
	for {
		msg := <-themix.inputc
		log.Println("msg: ", msg)
		themix.msgc[msg.Proposer] <- msg
	}
}
