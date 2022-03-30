package themix

import (
	"bytes"
	"log"
	"time"

	"themix.new.io/crypto/sha256"
	"themix.new.io/message/messagepb"
)

const maxround = 30

type instance struct {
	// TODO(chenzx): To be implemented.
	// ThemiX send message to instance through this channel.
	msgc         chan *messagepb.Msg
	outputc      chan *messagepb.Msg
	decideCh     chan []byte
	finishCh     chan []byte
	id           uint32
	n            int
	f            int
	round        int
	decided      bool
	proposal     *messagepb.Msg
	digest       []byte
	delta        int
	deltaBar     int
	hasEcho      bool
	numEcho      int
	hasReady     bool
	numReady     int
	bvalZero     []bool
	bvalOne      []bool
	numBvalZero  []int
	numBvalOne   []int
	zeroEndorsed []bool
	oneEndorsed  []bool
	hasSentAux   []bool
	numAuxZero   []int
	numAuxOne    []int
	numCoin      []int
	hasSentCoin  []bool
	startR       bool
	expireR      bool
	startB       []bool
	expireB      []bool
}

func initInstance(id uint32, n, f, delta, deltaBar int, msgc chan *messagepb.Msg, outputc chan *messagepb.Msg, decideCh, finishCh chan []byte) *instance {
	inst := &instance{
		msgc:         msgc,
		outputc:      outputc,
		decideCh:     decideCh,
		finishCh:     finishCh,
		id:           id,
		n:            n,
		f:            f,
		delta:        delta,
		deltaBar:     deltaBar,
		bvalZero:     make([]bool, maxround),
		bvalOne:      make([]bool, maxround),
		numBvalZero:  make([]int, maxround),
		numBvalOne:   make([]int, maxround),
		zeroEndorsed: make([]bool, maxround),
		oneEndorsed:  make([]bool, maxround),
		hasSentAux:   make([]bool, maxround),
		numAuxZero:   make([]int, maxround),
		numAuxOne:    make([]int, maxround),
		numCoin:      make([]int, maxround),
		hasSentCoin:  make([]bool, maxround),
		startB:       make([]bool, maxround),
		expireB:      make([]bool, maxround),
	}
	go inst.run()
	return inst
}

func (inst *instance) run() {
	for {
		select {
		case msg := <-inst.msgc:
			log.Printf("msg Type(%d), Proposer(%d), Seq(%d), From(%d)\n", msg.Type, msg.Proposer, msg.Seq, msg.From)
			inst.handleMsg(msg)
		case <-inst.finishCh:
			return
		}
	}
}

func (inst *instance) handleMsg(msg *messagepb.Msg) {
	switch msg.Type {
	case messagepb.MsgType_VAL:
		inst.proposal = msg
		digest, err := sha256.ComputeHash(msg.Content)
		if err != nil {
			log.Fatal("sha256.ComputeHash: ", err)
		}
		if inst.digest != nil {
			if !bytes.Equal(digest, inst.digest) {
				log.Fatal("bytes.Equal: receive different proposals")
			}
		} else {
			inst.digest = digest
		}
		if !inst.hasEcho {
			inst.hasEcho = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_ECHO,
				From:     inst.id,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Content:  digest,
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
		if inst.numReady >= inst.f+1 && inst.round == 0 &&
			!inst.bvalOne[inst.round] && inst.proposal != nil {
			inst.bvalOne[inst.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_BVAL,
				From:     inst.id,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Content:  []byte{1},
			}
			inst.outputc <- m
		}
	case messagepb.MsgType_ECHO:
		if inst.digest == nil {
			inst.digest = msg.Content
		} else if !bytes.Equal(msg.Content, inst.digest) {
			log.Fatal("bytes.Equal: receive different proposals")
		}
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
		if inst.numReady >= inst.f+1 && inst.round == 0 &&
			!inst.bvalOne[inst.round] && inst.proposal != nil {
			inst.bvalOne[inst.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_BVAL,
				From:     inst.id,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Content:  []byte{1},
			}
			inst.outputc <- m
		}
	case messagepb.MsgType_BVAL:
		switch msg.Content[0] {
		case 0:
			inst.numBvalZero[msg.Round]++
		case 1:
			inst.numBvalOne[msg.Round]++
		}
		if msg.Content[0] == 0 && inst.round == int(msg.Round) &&
			inst.numBvalZero[inst.round] >= inst.f+1 &&
			!inst.zeroEndorsed[inst.round] {
			inst.zeroEndorsed[inst.round] = true
			// TODO(chenzx): broadcast f+1 BVAL(b, r)
			if !inst.hasSentAux[inst.round] {
				inst.hasSentAux[inst.round] = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_AUX,
					From:     inst.id,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    msg.Round,
					Content:  []byte{0},
				}
				inst.outputc <- m
				if !inst.startB[inst.round] {
					inst.startB[inst.round] = true
					go func() {
						time.Sleep(time.Duration(inst.deltaBar) * time.Millisecond)
						inst.expireB[inst.round] = true
						if inst.numAuxZero[inst.round]+inst.numAuxOne[inst.round] >= inst.f+1 &&
							!inst.hasSentCoin[inst.round] {
							inst.hasSentCoin[inst.round] = true
							m := &messagepb.Msg{
								Type:     messagepb.MsgType_COIN,
								From:     inst.id,
								Proposer: msg.Proposer,
								Seq:      msg.Seq,
								Round:    msg.Round,
								Content:  []byte{1},
							}
							inst.outputc <- m
						}
					}()
				}
			}
		}
		if msg.Content[0] == 1 && inst.round == int(msg.Round) &&
			inst.numBvalOne[inst.round] >= inst.f+1 &&
			!inst.oneEndorsed[inst.round] {
			inst.oneEndorsed[inst.round] = true
			// TODO(chenzx): broadcast f+1 BVAL(b, r)
			if !inst.hasSentAux[inst.round] {
				inst.hasSentAux[inst.round] = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_AUX,
					From:     inst.id,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    msg.Round,
					Content:  []byte{1},
				}
				inst.outputc <- m
				if !inst.startB[inst.round] {
					inst.startB[inst.round] = true
					go func() {
						time.Sleep(time.Duration(inst.deltaBar) * time.Millisecond)
						inst.expireB[inst.round] = true
						if inst.numAuxZero[inst.round]+inst.numAuxOne[inst.round] >= inst.f+1 &&
							!inst.hasSentCoin[inst.round] {
							inst.hasSentCoin[inst.round] = true
							m := &messagepb.Msg{
								Type:     messagepb.MsgType_COIN,
								From:     inst.id,
								Proposer: msg.Proposer,
								Seq:      msg.Seq,
								Round:    msg.Round,
								Content:  []byte{1},
							}
							inst.outputc <- m
						}
					}()
				}
			}
		}
	case messagepb.MsgType_AUX:
		switch msg.Content[0] {
		case 0:
			inst.numAuxZero[msg.Round]++
		case 1:
			inst.numAuxOne[msg.Round]++
		}
		if inst.round == int(msg.Round) &&
			inst.numAuxZero[inst.round]+inst.numAuxOne[inst.round] >= inst.f+1 &&
			inst.expireB[inst.round] && !inst.hasSentCoin[inst.round] {
			//TODO(chenzx): to be fulfilled
			inst.hasSentCoin[inst.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_COIN,
				From:     inst.id,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Round:    msg.Round,
				Content:  []byte{1},
			}
			inst.outputc <- m
		}
	case messagepb.MsgType_COIN:
		inst.numCoin[msg.Round]++
		if inst.round == int(msg.Round) && inst.numCoin[inst.round] >= inst.f+1 &&
			!inst.decided {
			inst.decided = true
			// coin := 1
			inst.decideCh <- []byte{}
		}
	case messagepb.MsgType_CANVOTEZERO:
		if inst.round == 0 && !inst.bvalZero[inst.round] && !inst.bvalOne[inst.round] {
			inst.bvalZero[inst.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_BVAL,
				From:     inst.id,
				Proposer: inst.id,
				Seq:      msg.Seq,
				Round:    uint32(inst.round),
				Content:  []byte{0},
			}
			inst.outputc <- m
		}
	}
}
