package themix

import (
	"log"

	"themix.new.io/client/clientpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
)

type Themix struct {
	inputc    chan *messagepb.Msg
	outputc   chan *messagepb.Msg
	msgc      map[uint32]chan *messagepb.Msg
	finishCh  map[uint32]chan []byte
	reqc      chan *clientpb.Request
	repc      chan []byte
	decideCh  chan []byte
	statusCh  chan uint32
	instances []*instance
	seq       uint32
	id        uint32
	n         int
	f         int
	delta     int
	deltaBar  int
	decided   int
	finish    int
	proposed  bool
}

func initThemix(id, seq uint32, blsSig *bls.BlsSig, n, f, delta, deltaBar int, inputc chan *messagepb.Msg, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte, statusCh chan uint32) *Themix {
	themix := &Themix{
		inputc:    inputc,
		outputc:   outputc,
		reqc:      reqc,
		repc:      repc,
		statusCh:  statusCh,
		decideCh:  make(chan []byte, n),
		finishCh:  make(map[uint32]chan []byte),
		msgc:      make(map[uint32]chan *messagepb.Msg),
		instances: make([]*instance, n),
		seq:       seq,
		id:        id,
		n:         n,
		f:         f,
		delta:     delta,
		deltaBar:  deltaBar,
	}
	for i := 0; i < int(n); i++ {
		msgc := make(chan *messagepb.Msg, BUFFER)
		finishCh := make(chan []byte)
		themix.finishCh[uint32(i)] = finishCh
		themix.msgc[uint32(i)] = msgc
		themix.instances[i] = initInstance(uint32(i), uint32(themix.id), themix.seq, n, f, delta, deltaBar, blsSig, msgc, outputc, themix.decideCh, finishCh)
	}
	return themix
}

func (themix *Themix) run() {
	for {
		select {
		case msg := <-themix.inputc:
			if msg.Type == messagepb.MsgType_CANFINISH {
				themix.finish++
				if themix.finish == themix.n {
					themix.reqc = nil
					themix.repc = nil
					themix.statusCh <- themix.seq
					for i := 0; i < themix.n; i++ {
						themix.finishCh[uint32(i)] <- []byte{}
					}
					return
				}
				break
			}
			if msg.Type == messagepb.MsgType_VAL && msg.Proposer == themix.id && len(msg.Content) != 0 {
				themix.proposed = true
			}
			themix.msgc[msg.Proposer] <- msg
		case <-themix.decideCh:
			themix.decided++
			log.Println("themix decide number: ", themix.decided)
			if themix.decided == 1 {
				if !themix.proposed {
					themix.reqc <- &clientpb.Request{}
				}
			} else if themix.decided == themix.n-themix.f {
				m := &messagepb.Msg{
					Type: messagepb.MsgType_CANVOTEZERO,
				}
				for i := 0; i < themix.n; i++ {
					themix.msgc[uint32(i)] <- m
				}
			} else if themix.decided == themix.n {
				if themix.proposed {
					themix.repc <- []byte{}
				}
				themix.outputc <- &messagepb.Msg{
					Type: messagepb.MsgType_CANFINISH,
				}
			}
		}
	}
}
