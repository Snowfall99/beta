package themix

import "themix.new.io/common/messagepb"

type instance struct {
	// TODO(chenzx): To be implemented.
	// ThemiX send message to instance through this channel.
	msgc chan *messagepb.Msg
	n    uint32
}

func initInstance(msgc chan *messagepb.Msg, n uint32) {
	// TODO(chenzx): to be implemented.
	inst := &instance{
		msgc: msgc,
		n:    n,
	}
	go inst.run()
}

func (inst *instance) run() {

}
