package themix

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"

	"themix.new.io/crypto/sha256"
	bls "themix.new.io/crypto/themixBLS"
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
	proposer     uint32
	sequence     uint32
	blsSig       *bls.BlsSig
	n            int
	f            int
	round        int
	decided      bool
	proposal     *messagepb.Msg
	digest       []byte
	binVals      uint8
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
	coinMsgs     [][]*messagepb.Msg
	startR       bool
	expireR      bool
	startB       []bool
	expireB      []bool
}

func initInstance(id, proposer, sequence uint32, n, f, delta, deltaBar int, blsSig *bls.BlsSig, msgc chan *messagepb.Msg, outputc chan *messagepb.Msg, decideCh, finishCh chan []byte) *instance {
	inst := &instance{
		msgc:         msgc,
		outputc:      outputc,
		decideCh:     decideCh,
		finishCh:     finishCh,
		id:           id,
		proposer:     proposer,
		sequence:     sequence,
		blsSig:       blsSig,
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
		coinMsgs:     make([][]*messagepb.Msg, maxround),
		startB:       make([]bool, maxround),
		expireB:      make([]bool, maxround),
	}
	for i := 0; i < maxround; i++ {
		inst.coinMsgs[i] = make([]*messagepb.Msg, n)
	}
	go inst.run()
	return inst
}

func (inst *instance) run() {
	for {
		select {
		case msg := <-inst.msgc:
			log.Printf("ID(%d), msg Type(%d), Proposer(%d), Seq(%d), From(%d), Round(%d)\n", inst.id, msg.Type, msg.Proposer, msg.Seq, msg.From, msg.Round)
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
						inst.sendCoin()
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
						inst.sendCoin()
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
		if inst.round == int(msg.Round) && !inst.hasSentCoin[inst.round] {
			inst.sendCoin()
		}
	case messagepb.MsgType_COIN:
		inst.coinMsgs[msg.Round][msg.From] = msg
		inst.numCoin[msg.Round]++
		if inst.round == int(msg.Round) && inst.numCoin[inst.round] >= inst.f+1 &&
			!inst.decided {
			inst.newRound()
		}
	case messagepb.MsgType_CANVOTEZERO:
		if inst.round == 0 && !inst.bvalZero[inst.round] && !inst.bvalOne[inst.round] {
			inst.bvalZero[inst.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_BVAL,
				Proposer: inst.id,
				Seq:      inst.sequence,
				Round:    uint32(inst.round),
				Content:  []byte{0},
			}
			inst.outputc <- m
		}
	}
}

func (inst *instance) sendCoin() {
	if !inst.hasSentCoin[inst.round] && inst.expireB[inst.round] {
		if !inst.decided {
			if inst.oneEndorsed[inst.round] && inst.numAuxOne[inst.round] >= inst.f+1 {
				inst.binVals = 1
			} else if inst.zeroEndorsed[inst.round] && inst.numAuxZero[inst.round] >= inst.f+1 {
				inst.binVals = 0
			} else if inst.zeroEndorsed[inst.round] && inst.oneEndorsed[inst.round] &&
				inst.numAuxZero[inst.round]+inst.numAuxOne[inst.round] >= inst.f+1 {
				inst.binVals = 2
			} else {
				return
			}
		}
	} else {
		return
	}
	inst.hasSentCoin[inst.round] = true
	m := &messagepb.Msg{
		Type:     messagepb.MsgType_COIN,
		Proposer: inst.id,
		Seq:      inst.sequence,
		Round:    uint32(inst.round),
		Content:  inst.blsSig.Sign(inst.getCoinInfo()),
	}
	inst.outputc <- m
}

func (inst *instance) getCoinInfo() []byte {
	bsender := make([]byte, 8)
	binary.LittleEndian.PutUint64(bsender, uint64(inst.id))
	bseq := make([]byte, 8)
	binary.LittleEndian.PutUint64(bseq, uint64(inst.sequence))
	b := make([]byte, 17)
	b = append(b, bsender...)
	b = append(b, bseq...)
	b = append(b, uint8(inst.round))
	return b
}

func (inst *instance) newRound() {
	sigShares := make([][]byte, 0)
	for _, m := range inst.coinMsgs[inst.round] {
		if m != nil {
			sigShares = append(sigShares, m.Content)
		}
	}
	if len(sigShares) < inst.f+1 {
		return
	}
	coin := inst.blsSig.Recover(inst.getCoinInfo(), sigShares, inst.f+1, inst.n)
	var nextVote byte
	if coin[0]%2 == inst.binVals {
		if (inst.binVals == 1 && inst.proposal == nil) ||
			(inst.binVals == 0 && inst.proposal == nil && inst.proposer == inst.id) {
			return
		}
		inst.decided = true
		inst.decideCh <- []byte{inst.binVals}
		nextVote = inst.binVals
	} else if inst.binVals != 2 {
		nextVote = inst.binVals
	} else {
		nextVote = coin[0] % 2
	}
	inst.round++
	if nextVote == 0 {
		inst.bvalZero[inst.round] = true
	} else {
		inst.bvalOne[inst.round] = true
	}
	m := &messagepb.Msg{
		Type:     messagepb.MsgType_BVAL,
		Proposer: inst.id,
		Seq:      inst.sequence,
		Round:    uint32(inst.round),
		Content:  []byte{nextVote},
	}
	inst.outputc <- m
}
