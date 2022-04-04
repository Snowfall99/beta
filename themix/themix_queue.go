package themix

import (
	"themix.new.io/client/clientpb"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
)

type STATUS int

const (
	NIL STATUS = iota
	TOMBSTONE
	ALIVE
)

const BUFFER = 1024

type ThemixQue struct {
	inputc   chan *messagepb.Msg
	outputc  chan *messagepb.Msg
	reqc     chan *clientpb.Request
	repc     chan []byte
	statusCh chan uint32
	msgc     map[uint32]chan *messagepb.Msg
	queue    map[uint32]STATUS
	id       uint32
	blsSig   *bls.BlsSig
	n        int
	f        int
	delta    int
	deltaBar int
}

func initThemixQue(id uint32, blsSig *bls.BlsSig, n, f, delta, deltaBar int, inputc, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte) *ThemixQue {
	return &ThemixQue{
		inputc:   inputc,
		outputc:  outputc,
		reqc:     reqc,
		repc:     repc,
		statusCh: make(chan uint32, BUFFER),
		msgc:     make(map[uint32]chan *messagepb.Msg),
		queue:    make(map[uint32]STATUS),
		id:       id,
		blsSig:   blsSig,
		n:        n,
		f:        f,
		delta:    delta,
		deltaBar: deltaBar,
	}
}

func (themixQue *ThemixQue) run() {
	for {
		// Get msg from input channel.
		// Route msg to the right themix instance according to seq.
		// If related instance doesn't exist, create it.
		select {
		case seq := <-themixQue.statusCh:
			themixQue.queue[seq] = TOMBSTONE
			close(themixQue.msgc[seq])
		case msg := <-themixQue.inputc:
			if themixQue.queue[msg.Seq] == ALIVE && themixQue.msgc[msg.Seq] != nil {
				themixQue.msgc[msg.Seq] <- msg
			} else if themixQue.queue[msg.Seq] == TOMBSTONE {
				break
			} else {
				ch := make(chan *messagepb.Msg, BUFFER)
				themixQue.msgc[msg.Seq] = ch
				themix := initThemix(themixQue.id, msg.Seq, themixQue.blsSig, themixQue.n, themixQue.f, themixQue.delta, themixQue.deltaBar, ch, themixQue.outputc, themixQue.reqc, themixQue.repc, themixQue.statusCh)
				go themix.run()
				themixQue.queue[msg.Seq] = ALIVE
				themixQue.msgc[msg.Seq] <- msg
			}
		}
	}
}
