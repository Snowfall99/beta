package themix

import (
	"themix.new.io/client/clientpb"
	"themix.new.io/common/messagepb"
)

type ThemixQue struct {
	inputc   chan *messagepb.Msg
	outputc  chan *messagepb.Msg
	reqc     chan *clientpb.Request
	repc     chan []byte
	msgc     map[uint32]chan *messagepb.Msg
	id       uint32
	n        int
	f        int
	delta    int
	deltaBar int
}

func initThemixQue(id uint32, n, f, delta, deltaBar int, inputc, outputc chan *messagepb.Msg, reqc chan *clientpb.Request, repc chan []byte) *ThemixQue {
	return &ThemixQue{
		inputc:   inputc,
		outputc:  outputc,
		reqc:     reqc,
		repc:     repc,
		msgc:     make(map[uint32]chan *messagepb.Msg),
		id:       id,
		n:        n,
		f:        f,
		delta:    delta,
		deltaBar: deltaBar,
	}
}

func (themixQue *ThemixQue) run() {
	// TODO(chenzx): A thread pool to get msg from input channel and handle them.
	// This is supposed to be implemented as a for loop.
	for {
		// Get msg from input channel.
		// Route msg to the right themix instance according to seq.
		// If related instance doesn't exist, create it.
		msg := <-themixQue.inputc
		if themixQue.msgc[msg.Seq] != nil {
			themixQue.msgc[msg.Seq] <- msg
		} else {
			ch := make(chan *messagepb.Msg)
			themixQue.msgc[msg.Seq] = ch
			themix := initThemix(themixQue.id, themixQue.n, themixQue.f, themixQue.delta, themixQue.deltaBar, ch, themixQue.outputc, themixQue.reqc, themixQue.repc)
			go themix.run()
			themixQue.msgc[msg.Seq] <- msg
		}
	}
}
