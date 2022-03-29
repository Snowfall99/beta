package themix

import (
	"context"
	"log"
	"net"
	"runtime"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"themix.new.io/client/clientpb"
	"themix.new.io/message/messagepb"
)

type Proposer struct {
	reqc       chan *clientpb.Request
	outputc    chan *messagepb.Msg
	verifyReq  chan *clientpb.Payload
	verifyResp chan messagepb.VerifyResult
	id         uint32
	seq        uint32
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
	verifyReq := make(chan *clientpb.Payload, batchsize)
	verifyResp := make(chan messagepb.VerifyResult)
	proposer := &Proposer{
		reqc:       reqc,
		outputc:    outputc,
		verifyReq:  verifyReq,
		verifyResp: verifyResp,
		id:         id,
	}
	go proposer.initVerifyPool()
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

func (proposer *Proposer) initVerifyPool() {
	for i := 0; i < 2*runtime.NumCPU(); i++ {
		go func() {
			for {
				req := <-proposer.verifyReq
				if verify(req) {
					proposer.verifyResp <- messagepb.VerifyResult_ACCEPT
				} else {
					proposer.verifyResp <- messagepb.VerifyResult_REJECT
				}
			}
		}()
	}
	for {
		req := <-proposer.verifyReq
		if verify(req) {
			proposer.verifyResp <- messagepb.VerifyResult_ACCEPT
		} else {
			proposer.verifyResp <- messagepb.VerifyResult_REJECT
		}
	}
}

func (proposer *Proposer) run() {
	// TODO(chenzx): get requets from client.
	// Implement a thread pool for verfication client signatures.
	for {
		req := <-proposer.reqc
		for _, payload := range req.Payload {
			proposer.verifyReq <- payload
		}
		result := true
		for i := 0; i < len(req.Payload); i++ {
			resp := <-proposer.verifyResp
			if resp != messagepb.VerifyResult_ACCEPT {
				result = false
			}
		}
		if result {
			// TODO(chenzx): Send message to outputc after noise layer is completed.
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
}

func verify(req *clientpb.Payload) bool {
	// TODO(chenzx): To be implemented.
	return true
}
