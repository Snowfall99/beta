package themix

import (
	"themix.new.io/common/messagepb"
)

type ThemixQue struct {
	inputc  chan *messagepb.Msg
	outputc chan *messagepb.Msg
	repc    chan []byte
	msgc    map[uint32]chan *messagepb.Msg
	n       uint32
}

func initThemixQue(inputc, outputc chan *messagepb.Msg, repc chan []byte, n uint32) *ThemixQue {
	return &ThemixQue{
		inputc:  inputc,
		outputc: outputc,
		repc:    repc,
		msgc:    make(map[uint32]chan *messagepb.Msg),
		n:       n,
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
			themix := initThemix(ch, themixQue.repc, themixQue.n)
			go themix.run()
			themixQue.msgc[msg.Seq] <- msg
		}
	}
}
