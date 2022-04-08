package themix

import (
	"bytes"
	"log"
	"time"

	"themix.new.io/crypto/sha256"
	"themix.new.io/message/messagepb"
)

type rbcInstance struct {
	// ID of instance.
	id uint32
	// Number of all nodes.
	n int
	// Number of faulty nodes.
	f int
	// Original client's proposal.
	proposal *messagepb.Msg
	// Digest of original proposal.
	digest []byte
	// Timeout R.
	deltaBar int
	// Has already sent ECHO.
	hasEcho bool
	// Has already sent READY.
	hasReady bool
	// Number of received ECHO.
	numEcho int
	// Number of received READY.
	numReady int
	// Signature of READY from different nodes.
	readySign *messagepb.Collection
	// Whether RBC has deliver or not.
	deliver bool
	// Whether timeout R is start or not.
	startR bool
	// Has already timeout R.
	expireR bool

	// Channel used for sending message to transport layer.
	outputc chan *messagepb.Msg
	// Channel used for receiving message.
	msgc chan *messagepb.Msg
	// Channel used for delivering RBC result.
	deliverCh chan *messagepb.Msg
	// Finish channel is used for receiving finish message.
	finishCh chan []byte
}

func initRBC(id uint32, n, f, deltaBar int, msgc, outputc, deliverCh chan *messagepb.Msg, finishCh chan []byte) *rbcInstance {
	log.Println("rbc init")
	rbc := &rbcInstance{
		id:        id,
		n:         n,
		f:         f,
		deltaBar:  deltaBar,
		outputc:   outputc,
		msgc:      msgc,
		deliverCh: deliverCh,
		finishCh:  finishCh,
		readySign: &messagepb.Collection{Slot: make([][]byte, n)},
	}
	go rbc.run()
	return rbc
}

func (rbc *rbcInstance) run() {
	for {
		select {
		case msg := <-rbc.msgc:
			if len(msg.Content) > 0 {
				log.Printf("[rbc] ID(%d), Type(%s), Prposer(%d), Seq(%d), From(%d), Round(%d), Content(%d)\n",
					rbc.id, messagepb.MsgType_name[int32(msg.Type)], msg.Proposer, msg.Seq, msg.From, msg.Round, msg.Content[0])
			} else {
				log.Printf("[rbc] ID(%d), Type(%s), Prposer(%d), Seq(%d), From(%d), Round(%d)\n",
					rbc.id, messagepb.MsgType_name[int32(msg.Type)], msg.Proposer, msg.Seq, msg.From, msg.Round)
			}
			rbc.handleMsg(msg)
		case <-rbc.finishCh:
			log.Println("[rbc] finish")
			rbc.deliverCh = nil
			return
		}
	}
}

func (rbc *rbcInstance) handleMsg(msg *messagepb.Msg) {
	switch msg.Type {
	case messagepb.MsgType_VAL:
		rbc.proposal = msg
		digest, err := sha256.ComputeHash(msg.Content)
		if err != nil {
			log.Fatal("sha256.ComputeHash: ", err)
		}
		if rbc.digest != nil {
			if !bytes.Equal(digest, rbc.digest) {
				log.Fatal("bytes.Equal: receive different proposals")
			}
		} else {
			rbc.digest = digest
		}
		if !rbc.hasEcho {
			rbc.hasEcho = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_ECHO,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Content:  digest,
			}
			rbc.outputc <- m
		}
		if !rbc.startR {
			rbc.startR = true
			go func() {
				time.Sleep(time.Duration(rbc.deltaBar) * time.Millisecond)
				rbc.expireR = true
				if rbc.numEcho >= rbc.f+1 && !rbc.hasReady {
					rbc.hasReady = true
					m := &messagepb.Msg{
						Type:     messagepb.MsgType_READY,
						Proposer: msg.Proposer,
						Seq:      msg.Seq,
						Content:  msg.Content,
					}
					rbc.outputc <- m
				}
			}()
		}
		if rbc.numReady >= rbc.f+1 && rbc.proposal != nil && !rbc.deliver {
			rbc.deliver = true
			rbc.deliverCh <- rbc.proposal
		}
	case messagepb.MsgType_ECHO:
		if rbc.digest == nil {
			rbc.digest = msg.Content
		} else if !bytes.Equal(msg.Content, rbc.digest) {
			log.Fatal("bytes.Equal: receive different proposals")
		}
		rbc.numEcho++
		if rbc.numEcho >= rbc.f+1 && rbc.expireR && !rbc.hasReady {
			rbc.hasReady = true
			m := &messagepb.Msg{
				Type:     messagepb.MsgType_READY,
				Proposer: msg.Proposer,
				Seq:      msg.Seq,
				Content:  msg.Content,
			}
			rbc.outputc <- m
		}
	case messagepb.MsgType_READY:
		rbc.numReady++
		rbc.readySign.Slot[msg.From] = msg.Signature
		if rbc.numReady >= rbc.f+1 && rbc.proposal != nil && !rbc.deliver {
			rbc.deliver = true
			rbc.deliverCh <- rbc.proposal
			collection := serialCollection(rbc.readySign)
			rbc.outputc <- &messagepb.Msg{
				Type:       messagepb.MsgType_RCOLLECTION,
				Proposer:   msg.Proposer,
				Seq:        msg.Seq,
				Collection: collection,
			}
		}
	case messagepb.MsgType_RCOLLECTION:
		// if rbc.proposal == nil || rbc.deliver {
		// 	break
		// }
		// if !rbc.verifyRcollection(msg) {
		// 	log.Fatal("inst.verifyRcollection fail")
		// }
		// rbc.outputc <- msg
		// rbc.deliver = true
		// rbc.deliverCh <- rbc.proposal
		log.Println("RCOLLECTION is not implemented")
	default:
		log.Fatal("Undefined message type")
	}
}

func (rbc *rbcInstance) verifyRcollection(collection *messagepb.Msg) bool {
	return true
}
