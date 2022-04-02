package themix

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"

	"google.golang.org/protobuf/proto"
	"themix.new.io/crypto/sha256"
	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
)

const maxround = 30

type instance struct {
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
	readySign    *messagepb.Collection
	bvalZero     []bool
	bvalOne      []bool
	numBvalZero  []int
	numBvalOne   []int
	bzeroSign    []*messagepb.Collection
	boneSign     []*messagepb.Collection
	zeroEndorsed []bool
	oneEndorsed  []bool
	hasSentAux   []bool
	numAuxZero   []int
	numAuxOne    []int
	numCoin      []int
	hasSentCoin  []bool
	coinMsgs     [][]*messagepb.Msg
	coinResult   [][]byte
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
		readySign:    &messagepb.Collection{Slot: make([][]byte, n)},
		bzeroSign:    make([]*messagepb.Collection, maxround),
		boneSign:     make([]*messagepb.Collection, maxround),
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
		coinResult:   make([][]byte, maxround),
		startB:       make([]bool, maxround),
		expireB:      make([]bool, maxround),
	}
	for i := 0; i < maxround; i++ {
		inst.coinMsgs[i] = make([]*messagepb.Msg, n)
		inst.bzeroSign[i] = &messagepb.Collection{Slot: make([][]byte, n)}
		inst.boneSign[i] = &messagepb.Collection{Slot: make([][]byte, n)}
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
		inst.readySign.Slot[msg.From] = msg.Signature
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
			collection := serialCollection(inst.readySign)
			inst.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_RCOLLECTION,
				Proposer:   msg.Proposer,
				Seq:        msg.Seq,
				Collection: collection,
			}
		}
	case messagepb.MsgType_BVAL:
		switch msg.Content[0] {
		case 0:
			inst.numBvalZero[msg.Round]++
			inst.bzeroSign[msg.Round].Slot[msg.From] = msg.Signature
		case 1:
			inst.numBvalOne[msg.Round]++
			inst.boneSign[msg.From].Slot[msg.From] = msg.Signature
		}
		if msg.Content[0] == 0 && inst.round == int(msg.Round) &&
			inst.numBvalZero[inst.round] >= inst.f+1 &&
			!inst.zeroEndorsed[inst.round] {
			inst.zeroEndorsed[inst.round] = true
			collection := serialCollection(inst.bzeroSign[msg.Round])
			inst.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_BZEROCOLLECTION,
				Proposer:   msg.Proposer,
				Round:      msg.Round,
				Seq:        msg.Seq,
				Collection: collection,
			}
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
			collection := serialCollection(inst.boneSign[msg.Round])
			inst.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_BONECOLLECTION,
				Proposer:   msg.Proposer,
				Round:      msg.Round,
				Seq:        msg.Seq,
				Collection: collection,
			}
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
		if msg.Content[0] == 0 && inst.round == (int(msg.Round)+1) &&
			inst.coinResult[msg.Round][0] == 0 && inst.numBvalZero[msg.Round] >= inst.f+1 {
			if !inst.bvalZero[inst.round] || !inst.hasSentAux[inst.round] {
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_BVAL,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    uint32(inst.round),
					Content:  []byte{0},
				}
				inst.outputc <- m
			}
		}
		if msg.Content[0] == 1 && inst.round == (int(msg.Round)+1) &&
			inst.coinResult[msg.Round][0] == 1 && inst.numBvalOne[msg.Round] >= inst.f+1 {
			if !inst.bvalOne[inst.round] || !inst.hasSentAux[inst.round] {
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_BVAL,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    uint32(inst.round),
					Content:  []byte{1},
				}
				inst.outputc <- m
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
	case messagepb.MsgType_RCOLLECTION:
		if inst.round != 0 || inst.proposal == nil || inst.bvalZero[inst.round] || inst.bvalOne[inst.round] {
			break
		}
		if !inst.verifyRcollection(msg) {
			log.Fatal("inst.verifyRcollection fail")
		}
		inst.outputc <- msg
		inst.bvalOne[inst.round] = true
		m := &messagepb.Msg{
			Type:     messagepb.MsgType_BVAL,
			Proposer: msg.Proposer,
			Seq:      msg.Seq,
			Content:  []byte{1},
		}
		inst.outputc <- m
	case messagepb.MsgType_BZEROCOLLECTION:
		if inst.zeroEndorsed[msg.Round] || inst.oneEndorsed[msg.Round] || inst.hasSentAux[msg.Round] {
			break
		}
		if !inst.verifyBzero(msg) {
			log.Fatal("inst.verifyBzero fail")
		}
		inst.outputc <- msg
		inst.zeroEndorsed[msg.Round] = true
		inst.hasSentAux[msg.Round] = true
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
	case messagepb.MsgType_BONECOLLECTION:
		if inst.zeroEndorsed[msg.Round] || inst.oneEndorsed[msg.Round] || inst.hasSentAux[msg.Round] {
			break
		}
		if !inst.verifyBone(msg) {
			log.Fatal("inst.verifyBone fail")
		}
		inst.outputc <- msg
		inst.oneEndorsed[msg.Round] = true
		inst.hasSentAux[msg.Round] = true
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
	default:
		log.Fatal("Message's type is undefined")
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
	inst.coinResult[inst.round] = []byte{coin[0] % 2}
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

func serialCollection(collection *messagepb.Collection) []byte {
	data, err := proto.Marshal(collection)
	if err != nil {
		log.Fatal("proto.Marshal: ", err)
	}
	return data
}

func deserialCollection(data []byte) *messagepb.Collection {
	collection := &messagepb.Collection{}
	err := proto.Unmarshal(data, collection)
	if err != nil {
		log.Fatal("proto.Unmarshal: ", err)
	}
	return collection
}

func (inst *instance) verifyRcollection(collection *messagepb.Msg) bool {
	return true
}

func (inst *instance) verifyBzero(collection *messagepb.Msg) bool {
	return true
}

func (inst *instance) verifyBone(collection *messagepb.Msg) bool {
	return true
}
