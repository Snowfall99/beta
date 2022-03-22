package themix

import (
	"log"

	"themix.new.io/client/clientpb"
	"themix.new.io/common/messagepb"
)

type Themix struct {
	// TODO(chenzx): To be implemented.
	inputc   chan *messagepb.Msg
	outputc  chan *messagepb.Msg
	msgc     map[uint32]chan *messagepb.Msg
	reqc     chan *clientpb.Request
	repc     chan []byte
	decideCh chan []byte
	id       uint32
	n        int
	f        int
	decided  int
	proposed bool
}

func initThemix(inputc chan *messagepb.Msg, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte, n, f int, id uint32) *Themix {
	decideCh := make(chan []byte)
	themix := &Themix{
		inputc:   inputc,
		outputc:  outputc,
		reqc:     reqc,
		repc:     repc,
		decideCh: decideCh,
		msgc:     make(map[uint32]chan *messagepb.Msg),
		id:       id,
		n:        n,
		f:        f,
	}
	for i := 0; i < int(n); i++ {
		msgc := make(chan *messagepb.Msg)
		themix.msgc[uint32(i)] = msgc
		go initInstance(msgc, outputc, decideCh, n, f, uint32(i))
	}
	return themix
}

func (themix *Themix) run() {
	go func() {
		for {
			<-themix.decideCh
			themix.decided++
			log.Println("themix decide number: ", themix.decided)
			if themix.decided == 1 {
				if !themix.proposed {
					themix.reqc <- &clientpb.Request{}
				}
			} else if themix.decided == themix.n {
				themix.repc <- []byte{}
				themix.reqc = nil
				themix.repc = nil
				break
			}
		}
	}()
	for {
		msg := <-themix.inputc
		if msg.Type == messagepb.MsgType_VAL && msg.Proposer == themix.id {
			themix.proposed = true
		}
		// TODO(chenzx): this thread should exit after decide listen thread
		// if themix.decided == themix.n {
		// 	break
		// }
		themix.msgc[msg.Proposer] <- msg
	}
}
