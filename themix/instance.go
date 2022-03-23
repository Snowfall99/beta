package themix

import (
	"log"
	"time"

	"themix.new.io/common/messagepb"
)

type instance struct {
	// TODO(chenzx): To be implemented.
	// ThemiX send message to instance through this channel.
	msgc     chan *messagepb.Msg
	outputc  chan *messagepb.Msg
	decideCh chan []byte
	id       uint32
	n        int
	f        int
	proposal *messagepb.Msg
	delta    int
	deltaBar int
	hasEcho  bool
	numEcho  int
	hasReady bool
	numReady int
	startR   bool
	expireR  bool
}

func initInstance(id uint32, n, f, delta, deltaBar int, msgc chan *messagepb.Msg, outputc chan *messagepb.Msg, decideCh chan []byte) {
	// TODO(chenzx): to be implemented.
	inst := &instance{
		msgc:     msgc,
		outputc:  outputc,
		decideCh: decideCh,
		id:       id,
		n:        n,
		f:        f,
		delta:    delta,
		deltaBar: deltaBar,
	}
	go inst.run()
}

func (inst *instance) run() {
	for {
		msg := <-inst.msgc
		log.Printf("msg Type(%d), Proposer(%d), Seq(%d), From(%d)\n", msg.Type, msg.Proposer, msg.Seq, msg.From)
		switch msg.Type {
		case messagepb.MsgType_VAL:
			inst.proposal = msg
			if !inst.hasEcho {
				inst.hasEcho = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_ECHO,
					From:     inst.id,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Content:  msg.Content,
				}
				inst.outputc <- m
			}
			if !inst.startR {
				inst.startR = true
				go func() {
					time.Sleep(time.Duration(inst.deltaBar) * time.Millisecond)
					inst.expireR = true
					if inst.numEcho >= inst.f+1 && !inst.hasReady {
						inst.hasReady = true
						m := &messagepb.Msg{
							Type:     messagepb.MsgType_READY,
							From:     inst.id,
							Proposer: msg.Proposer,
							Seq:      msg.Seq,
							Content:  msg.Content,
						}
						inst.outputc <- m
					}
				}()
			}
		case messagepb.MsgType_ECHO:
			inst.numEcho++
			if inst.numEcho >= inst.f+1 && inst.expireR && !inst.hasReady {
				inst.hasReady = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_READY,
					From:     inst.id,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Content:  msg.Content,
				}
				inst.outputc <- m
			}
		case messagepb.MsgType_READY:
			inst.numReady++
			if inst.numReady == inst.f+1 {
				inst.decideCh <- []byte{}
			}
		}
	}
}
