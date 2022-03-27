package themix

import (
	"log"

	"themix.new.io/client/clientpb"
	"themix.new.io/message/messagepb"
)

type Themix struct {
	inputc    chan *messagepb.Msg
	outputc   chan *messagepb.Msg
	msgc      map[uint32]chan *messagepb.Msg
	reqc      chan *clientpb.Request
	repc      chan []byte
	decideCh  chan []byte
	instances []*instance
	id        uint32
	n         int
	f         int
	delta     int
	deltaBar  int
	decided   int
	proposed  bool
}

func initThemix(id uint32, n, f, delta, deltaBar int, inputc chan *messagepb.Msg, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte) *Themix {
	decideCh := make(chan []byte)
	themix := &Themix{
		inputc:    inputc,
		outputc:   outputc,
		reqc:      reqc,
		repc:      repc,
		decideCh:  decideCh,
		msgc:      make(map[uint32]chan *messagepb.Msg),
		instances: make([]*instance, n),
		id:        id,
		n:         n,
		f:         f,
		delta:     delta,
		deltaBar:  deltaBar,
	}
	for i := 0; i < int(n); i++ {
		msgc := make(chan *messagepb.Msg)
		themix.msgc[uint32(i)] = msgc
		themix.instances[i] = initInstance(uint32(i), n, f, delta, deltaBar, msgc, outputc, decideCh)
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
