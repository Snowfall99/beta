package themix

import (
	"crypto/ecdsa"
	"log"

	"google.golang.org/protobuf/proto"
	"themix.new.io/client/clientpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
)

type decideValue struct {
	value uint8
	from  uint32
}

type Themix struct {
	inputc       chan *messagepb.Msg
	outputc      chan *messagepb.Msg
	rbcMsgc      map[uint32]chan *messagepb.Msg
	abaMsgc      map[uint32]chan *messagepb.Msg
	rbcFinishCh  map[uint32]chan []byte
	abaFinishCh  map[uint32]chan []byte
	pubkeys      map[uint32]*ecdsa.PublicKey
	reqc         chan *clientpb.Request
	repc         chan []byte
	decideCh     chan decideValue
	statusCh     chan uint32
	rbcInstances []*rbcInstance
	abaInstances []*abaInstance
	seq          uint32
	id           uint32
	n            int
	f            int
	delta        int
	deltaBar     int
	decided      int
	finish       int
	proposal     *messagepb.Msg
	decideValue  []byte
}

func initThemix(id, seq uint32, blsSig *bls.BlsSig, n, f, delta, deltaBar int, inputc chan *messagepb.Msg, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte, statusCh chan uint32, pubkeys map[uint32]*ecdsa.PublicKey) *Themix {
	themix := &Themix{
		inputc:       inputc,
		outputc:      outputc,
		reqc:         reqc,
		repc:         repc,
		statusCh:     statusCh,
		decideCh:     make(chan decideValue, n),
		rbcFinishCh:  make(map[uint32]chan []byte),
		abaFinishCh:  make(map[uint32]chan []byte),
		rbcMsgc:      make(map[uint32]chan *messagepb.Msg),
		abaMsgc:      make(map[uint32]chan *messagepb.Msg),
		rbcInstances: make([]*rbcInstance, n),
		abaInstances: make([]*abaInstance, n),
		seq:          seq,
		id:           id,
		n:            n,
		f:            f,
		delta:        delta,
		deltaBar:     deltaBar,
		decideValue:  make([]byte, n),
		pubkeys:      pubkeys,
	}
	for i := 0; i < int(n); i++ {
		rbcMsgc := make(chan *messagepb.Msg, BUFFER)
		abaMsgc := make(chan *messagepb.Msg, BUFFER)
		rbcFinishCh := make(chan []byte)
		abaFinishCh := make(chan []byte)
		deliverCh := make(chan *messagepb.Msg)
		themix.rbcFinishCh[uint32(i)] = rbcFinishCh
		themix.abaFinishCh[uint32(i)] = abaFinishCh
		themix.rbcMsgc[uint32(i)] = rbcMsgc
		themix.abaMsgc[uint32(i)] = abaMsgc
		themix.rbcInstances[i] = initRBC(uint32(i), themix.n, themix.f, themix.deltaBar, rbcMsgc, outputc, deliverCh, rbcFinishCh, themix.pubkeys)
		themix.abaInstances[i] = initABA(uint32(i), id, themix.seq, themix.n, themix.f, themix.deltaBar, blsSig, abaMsgc, outputc, deliverCh, themix.decideCh, abaFinishCh, themix.pubkeys)
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
						themix.rbcFinishCh[uint32(i)] <- []byte{}
						themix.abaFinishCh[uint32(i)] <- []byte{}
					}
					return
				}
				break
			}
			if msg.Type == messagepb.MsgType_VAL && msg.Proposer == themix.id && len(msg.Content) != 0 {
				themix.proposal = msg
			}
			if msg.Type == messagepb.MsgType_VAL || msg.Type == messagepb.MsgType_ECHO || msg.Type == messagepb.MsgType_READY || msg.Type == messagepb.MsgType_RCOLLECTION {
				themix.rbcMsgc[msg.Proposer] <- msg
			} else {
				themix.abaMsgc[msg.Proposer] <- msg
			}
		case decideValue := <-themix.decideCh:
			themix.decideValue[decideValue.from] = decideValue.value
			themix.decided++
			log.Println("themix decide number: ", themix.decided)
			if themix.decided == 1 {
				if themix.proposal == nil {
					themix.reqc <- &clientpb.Request{}
				}
			} else if themix.decided == themix.n-themix.f {
				m := &messagepb.Msg{
					Type: messagepb.MsgType_CANVOTEZERO,
					Seq:  themix.seq,
				}
				for i := 0; i < themix.n; i++ {
					themix.abaMsgc[uint32(i)] <- m
				}
			} else if themix.decided == themix.n {
				if themix.proposal != nil && themix.decideValue[themix.id] == 1 {
					themix.repc <- []byte{}
				} else if themix.proposal != nil {
					log.Println("Reproposal")
					req := &clientpb.Request{}
					err := proto.Unmarshal(themix.proposal.Content, req)
					if err != nil {
						log.Fatal("proto.Unmarshal: ", err)
					}
					themix.reqc <- req
				}
				themix.outputc <- &messagepb.Msg{
					Type: messagepb.MsgType_CANFINISH,
					Seq:  themix.seq,
				}
			}
		}
	}
}
