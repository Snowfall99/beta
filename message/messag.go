package message

import (
	"encoding/binary"

	"themix.new.io/message/messagepb"
)

func GetMsgInfo(msg *messagepb.Msg) []byte {
	btype := make([]byte, 8)
	binary.LittleEndian.PutUint64(btype, uint64(msg.Type))
	bseq := make([]byte, 8)
	binary.LittleEndian.PutUint64(bseq, uint64(msg.Seq))
	bproposer := make([]byte, 8)
	binary.LittleEndian.PutUint64(bproposer, uint64(msg.Proposer))
	b := make([]byte, 26)
	b = append(b, btype...)
	b = append(b, bseq...)
	b = append(b, bproposer...)
	b = append(b, uint8(msg.Round))
	if len(msg.Content) > 0 {
		b = append(b, uint8(msg.Content[0]))
	} else {
		b = append(b, uint8(0))
	}
	return b
}
