package themix

import (
	"encoding/binary"
	"log"
	"sync"
	"time"

	bls "themix.new.io/crypto/themixBLS"
	"themix.new.io/message/messagepb"
)

type abaInstance struct {
	msgc         chan *messagepb.Msg
	outputc      chan *messagepb.Msg
	deliverCh    chan *messagepb.Msg
	finishCh     chan []byte
	decideCh     chan []byte
	id           uint32
	proposer     uint32
	sequence     uint32
	blsSig       *bls.BlsSig
	n            int
	f            int
	round        int
	decided      bool
	proposal     *messagepb.Msg
	binVals      uint8
	deltaBar     int
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
	startB       []bool
	expireB      []bool
	lock         sync.Mutex
}

func initABA(id, proposer, sequence uint32, n, f, deltaBar int, blsSig *bls.BlsSig, msgc, outputc, deliverCh chan *messagepb.Msg, deciedeCh, finishCh chan []byte) *abaInstance {
	log.Println("aba init")
	aba := &abaInstance{
		msgc:         msgc,
		outputc:      outputc,
		deliverCh:    deliverCh,
		decideCh:     deciedeCh,
		finishCh:     finishCh,
		id:           id,
		proposer:     proposer,
		sequence:     sequence,
		blsSig:       blsSig,
		n:            n,
		f:            f,
		deltaBar:     deltaBar,
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
		lock:         sync.Mutex{},
	}
	for i := 0; i < maxround; i++ {
		aba.coinMsgs[i] = make([]*messagepb.Msg, n)
		aba.bzeroSign[i] = &messagepb.Collection{Slot: make([][]byte, n)}
		aba.boneSign[i] = &messagepb.Collection{Slot: make([][]byte, n)}
	}
	go aba.run()
	return aba
}

func (aba *abaInstance) run() {
	for {
		select {
		case msg := <-aba.msgc:
			log.Printf("[aba] ID(%d), Type(%s), Prposer(%d), Seq(%d), From(%d), Round(%d)\n",
				aba.id, messagepb.MsgType_name[int32(msg.Type)], msg.Proposer, msg.Seq, msg.From, msg.Round)
			aba.handleMsg(msg)
		case proposal := <-aba.deliverCh:
			aba.proposal = proposal
			aba.propose()
		case <-aba.finishCh:
			log.Println("[aba] finish")
			aba.deliverCh = nil
			return
		}
	}
}

func (aba *abaInstance) propose() {
	if aba.round == 0 && !aba.bvalOne[aba.round] && aba.proposal != nil {
		aba.bvalOne[aba.round] = true
		m := &messagepb.Msg{
			Type:     messagepb.MsgType_BVAL,
			Proposer: aba.id,
			Seq:      aba.sequence,
			Content:  []byte{1},
		}
		aba.outputc <- m
	}
}

func (aba *abaInstance) handleMsg(msg *messagepb.Msg) {
	switch msg.Type {
	case messagepb.MsgType_BVAL:
		switch msg.Content[0] {
		case 0:
			aba.numBvalZero[msg.Round]++
			aba.bzeroSign[msg.Round].Slot[msg.From] = msg.Signature
		case 1:
			aba.numBvalOne[msg.Round]++
			aba.boneSign[msg.From].Slot[msg.From] = msg.Signature
		}
		if msg.Content[0] == 0 && aba.round == int(msg.Round) &&
			aba.numBvalZero[aba.round] >= aba.f+1 &&
			!aba.zeroEndorsed[aba.round] {
			aba.zeroEndorsed[aba.round] = true
			collection := serialCollection(aba.bzeroSign[msg.Round])
			aba.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_BZEROCOLLECTION,
				Proposer:   msg.Proposer,
				Round:      msg.Round,
				Seq:        msg.Seq,
				Collection: collection,
			}
			if !aba.hasSentAux[aba.round] {
				aba.hasSentAux[aba.round] = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_AUX,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    msg.Round,
					Content:  []byte{0},
				}
				aba.outputc <- m
				if !aba.startB[aba.round] {
					aba.startB[aba.round] = true
					go func() {
						time.Sleep(time.Duration(aba.deltaBar) * time.Millisecond)
						aba.expireB[aba.round] = true
						aba.lock.Lock()
						aba.sendCoin()
						aba.lock.Unlock()
					}()
				}
			}
		}
		if msg.Content[0] == 1 && aba.round == int(msg.Round) &&
			aba.numBvalOne[aba.round] >= aba.f+1 &&
			!aba.oneEndorsed[aba.round] {
			aba.oneEndorsed[aba.round] = true
			collection := serialCollection(aba.boneSign[msg.Round])
			aba.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_BONECOLLECTION,
				Proposer:   msg.Proposer,
				Round:      msg.Round,
				Seq:        msg.Seq,
				Collection: collection,
			}
			if !aba.hasSentAux[aba.round] {
				aba.hasSentAux[aba.round] = true
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_AUX,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    msg.Round,
					Content:  []byte{1},
				}
				aba.outputc <- m
				if !aba.startB[aba.round] {
					aba.startB[aba.round] = true
					go func() {
						time.Sleep(time.Duration(aba.deltaBar) * time.Millisecond)
						aba.expireB[aba.round] = true
						aba.lock.Lock()
						aba.sendCoin()
						aba.lock.Unlock()
					}()
				}
			}
		}
		if msg.Content[0] == 0 && aba.round == (int(msg.Round)+1) &&
			aba.coinResult[msg.Round][0] == 0 && aba.numBvalZero[msg.Round] >= aba.f+1 {
			if !aba.bvalZero[aba.round] || !aba.hasSentAux[aba.round] {
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_BVAL,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    uint32(aba.round),
					Content:  []byte{0},
				}
				aba.outputc <- m
			}
		}
		if msg.Content[0] == 1 && aba.round == (int(msg.Round)+1) &&
			aba.coinResult[msg.Round][0] == 1 && aba.numBvalOne[msg.Round] >= aba.f+1 {
			if !aba.bvalOne[aba.round] || !aba.hasSentAux[aba.round] {
				m := &messagepb.Msg{
					Type:     messagepb.MsgType_BVAL,
					Proposer: msg.Proposer,
					Seq:      msg.Seq,
					Round:    uint32(aba.round),
					Content:  []byte{1},
				}
				aba.outputc <- m
			}
		}
	case messagepb.MsgType_AUX:
		switch msg.Content[0] {
		case 0:
			aba.numAuxZero[msg.Round]++
		case 1:
			aba.numAuxOne[msg.Round]++
		}
		if aba.round == int(msg.Round) && !aba.hasSentCoin[aba.round] {
			aba.lock.Lock()
			aba.sendCoin()
			aba.lock.Unlock()
		}
	case messagepb.MsgType_COIN:
		aba.coinMsgs[msg.Round][msg.From] = msg
		aba.numCoin[msg.Round]++
		if aba.round == int(msg.Round) && aba.numCoin[aba.round] >= aba.f+1 &&
			!aba.decided {
			aba.newRound()
		}
	case messagepb.MsgType_CANVOTEZERO:
		if aba.round == 0 && !aba.bvalZero[aba.round] && !aba.bvalOne[aba.round] {
			aba.bvalZero[aba.round] = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_BVAL,
				Proposer: aba.id,
				Seq:      aba.sequence,
				Round:    uint32(aba.round),
				Content:  []byte{0},
			}
			aba.outputc <- m
		}
	case messagepb.MsgType_BZEROCOLLECTION:
		if aba.zeroEndorsed[msg.Round] || aba.oneEndorsed[msg.Round] || aba.hasSentAux[msg.Round] {
			break
		}
		if !aba.verifyBzero(msg) {
			log.Fatal("aba.verifyBzero fail")
		}
		aba.outputc <- msg
		aba.zeroEndorsed[msg.Round] = true
		aba.hasSentAux[msg.Round] = true
		m := &messagepb.Msg{
			Type:     messagepb.MsgType_AUX,
			Proposer: msg.Proposer,
			Seq:      msg.Seq,
			Round:    msg.Round,
			Content:  []byte{0},
		}
		aba.outputc <- m
		if !aba.startB[aba.round] {
			aba.startB[aba.round] = true
			go func() {
				time.Sleep(time.Duration(aba.deltaBar) * time.Millisecond)
				aba.expireB[aba.round] = true
				aba.lock.Lock()
				aba.sendCoin()
				aba.lock.Unlock()
			}()
		}
	case messagepb.MsgType_BONECOLLECTION:
		if aba.zeroEndorsed[msg.Round] || aba.oneEndorsed[msg.Round] || aba.hasSentAux[msg.Round] {
			break
		}
		if !aba.verifyBone(msg) {
			log.Fatal("aba.verifyBone fail")
		}
		aba.outputc <- msg
		aba.oneEndorsed[msg.Round] = true
		aba.hasSentAux[msg.Round] = true
		m := &messagepb.Msg{
			Type:     messagepb.MsgType_AUX,
			Proposer: msg.Proposer,
			Seq:      msg.Seq,
			Round:    msg.Round,
			Content:  []byte{1},
		}
		aba.outputc <- m
		if !aba.startB[aba.round] {
			aba.startB[aba.round] = true
			go func() {
				time.Sleep(time.Duration(aba.deltaBar) * time.Millisecond)
				aba.expireB[aba.round] = true
				aba.lock.Lock()
				aba.sendCoin()
				aba.lock.Unlock()
			}()
		}
	default:
		log.Fatal("Message's type is undefined")
	}
}

func (aba *abaInstance) sendCoin() {
	if !aba.hasSentCoin[aba.round] && aba.expireB[aba.round] {
		if !aba.decided {
			if aba.oneEndorsed[aba.round] && aba.numAuxOne[aba.round] >= aba.f+1 {
				aba.binVals = 1
			} else if aba.zeroEndorsed[aba.round] && aba.numAuxZero[aba.round] >= aba.f+1 {
				aba.binVals = 0
			} else if aba.zeroEndorsed[aba.round] && aba.oneEndorsed[aba.round] &&
				aba.numAuxZero[aba.round]+aba.numAuxOne[aba.round] >= aba.f+1 {
				aba.binVals = 2
			} else {
				return
			}
		}
	} else {
		return
	}
	aba.hasSentCoin[aba.round] = true
	m := &messagepb.Msg{
		Type:     messagepb.MsgType_COIN,
		Proposer: aba.id,
		Seq:      aba.sequence,
		Round:    uint32(aba.round),
		Content:  aba.blsSig.Sign(aba.getCoinInfo()),
	}
	aba.outputc <- m
}

func (aba *abaInstance) getCoinInfo() []byte {
	bsender := make([]byte, 8)
	binary.LittleEndian.PutUint64(bsender, uint64(aba.id))
	bseq := make([]byte, 8)
	binary.LittleEndian.PutUint64(bseq, uint64(aba.sequence))
	b := make([]byte, 17)
	b = append(b, bsender...)
	b = append(b, bseq...)
	b = append(b, uint8(aba.round))
	return b
}

func (aba *abaInstance) newRound() {
	sigShares := make([][]byte, 0)
	for _, m := range aba.coinMsgs[aba.round] {
		if m != nil {
			sigShares = append(sigShares, m.Content)
		}
	}
	if len(sigShares) < aba.f+1 {
		return
	}
	coin := aba.blsSig.Recover(aba.getCoinInfo(), sigShares, aba.f+1, aba.n)
	aba.coinResult[aba.round] = []byte{coin[0] % 2}
	var nextVote byte
	if coin[0]%2 == aba.binVals {
		if (aba.binVals == 1 && aba.proposal == nil) ||
			(aba.binVals == 0 && aba.proposal == nil && aba.proposer == aba.id) {
			return
		}
		aba.decided = true
		log.Printf("aba %d decide", aba.id)
		aba.decideCh <- []byte{aba.binVals}
		nextVote = aba.binVals
	} else if aba.binVals != 2 {
		nextVote = aba.binVals
	} else {
		nextVote = coin[0] % 2
	}
	aba.round++
	if nextVote == 0 {
		aba.bvalZero[aba.round] = true
	} else {
		aba.bvalOne[aba.round] = true
	}
	m := &messagepb.Msg{
		Type:     messagepb.MsgType_BVAL,
		Proposer: aba.id,
		Seq:      aba.sequence,
		Round:    uint32(aba.round),
		Content:  []byte{nextVote},
	}
	aba.outputc <- m
}

func (aba *abaInstance) verifyBzero(collection *messagepb.Msg) bool {
	return true
}

func (aba *abaInstance) verifyBone(collection *messagepb.Msg) bool {
	return true
}
