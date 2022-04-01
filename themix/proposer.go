package themix

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"themix.new.io/client/clientpb"
	"themix.new.io/message/messagepb"
)

type Proposer struct {
	reqc    chan *clientpb.Request
	outputc chan *messagepb.Msg
	id      uint32
	seq     uint32
}

type server struct {
	clientpb.UnimplementedThemixServer
	reqc chan *clientpb.Request
	repc chan []byte
}

func (s *server) Post(ctx context.Context, req *clientpb.Request) (*clientpb.Response, error) {
	log.Println("receive request")
	s.reqc <- req
	<-s.repc
	log.Println("reply")
	return &clientpb.Response{Ok: true}, nil
}

// Init verify pool for client message's signature verification
// Setup http handler to receive client requests.
func initProposer(batchsize int, client string, reqc chan *clientpb.Request, repc chan []byte, outputc chan *messagepb.Msg, id uint32) {
	proposer := &Proposer{
		reqc:    reqc,
		outputc: outputc,
		id:      id,
	}
	go proposer.run()
	listener, err := net.Listen("tcp", client)
	if err != nil {
		log.Fatal("[proposer] net.Listen: ", err)
	}
	s := grpc.NewServer()
	clientpb.RegisterThemixServer(s, &server{reqc: reqc, repc: repc})
	err = s.Serve(listener)
	if err != nil {
		log.Fatal("[proposer] s.Serve: ", err)
	}
}

func (proposer *Proposer) run() {
	for {
		req := <-proposer.reqc
		data, err := proto.Marshal(req)
		if err != nil {
			log.Fatal("[proposer] proto.Marshal: ", err)
		}
		proposer.seq++
		msg := &messagepb.Msg{
			Type:     messagepb.MsgType_VAL,
			Proposer: proposer.id,
			Seq:      proposer.seq,
			Content:  data,
		}
		log.Printf("[proposer] propose: proposer(%d), seq(%d)\n", proposer.id, proposer.seq)
		proposer.outputc <- msg
	}
}
