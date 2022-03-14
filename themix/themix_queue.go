package themix

import (
	"log"

	"themix.new.io/common/messagepb"
)

type ThemixQue struct {
	inputc  chan *messagepb.Msg
	outputc chan *messagepb.Msg
	repc    chan []byte
	msgc    map[uint64]*messagepb.Msg
	queue   map[uint64]*Themix
}

func initThemixQue(inputc, outputc chan *messagepb.Msg, repc chan []byte) *ThemixQue {
	return &ThemixQue{
		inputc:  inputc,
		outputc: outputc,
		repc:    repc,
		msgc:    make(map[uint64]*messagepb.Msg),
		queue:   make(map[uint64]*Themix),
	}
}

func (themixQue *ThemixQue) run() {
	// TODO(chenzx): A thread pool to get msg from input channel and handle them.
	// This is supposed to be implemented as a for loop.
	for {
		<-themixQue.inputc
		themixQue.repc <- []byte{}
		log.Println("reply to client")
	}
}
