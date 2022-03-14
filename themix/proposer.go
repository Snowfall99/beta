package themix

import (
	"context"
	"log"
	"net"
	"runtime"

	"google.golang.org/grpc"
	"themix.new.io/client/clientpb"
	"themix.new.io/common/messagepb"
)

type Proposer struct {
	reqc       chan *clientpb.Request
	outputc    chan *messagepb.Msg
	verifyReq  chan *clientpb.Payload
	verifyResp chan messagepb.VerifyResult
}

type server struct {
	clientpb.UnimplementedThemixServer
	reqc chan *clientpb.Request
	repc chan []byte
}

func (s *server) Post(ctx context.Context, req *clientpb.Request) (*clientpb.Response, error) {
	s.reqc <- req
	<-s.repc
	return &clientpb.Response{Ok: true}, nil
}

// Init verify pool for client message's signature verification
// Setup http handler to receive client requests.
func initProposer(batchsize uint32, client string, reqc chan *clientpb.Request, repc chan []byte, outputc chan *messagepb.Msg) {
	verifyReq := make(chan *clientpb.Payload, batchsize)
	verifyResp := make(chan messagepb.VerifyResult)
	proposer := &Proposer{
		reqc:       reqc,
		outputc:    outputc,
		verifyReq:  verifyReq,
		verifyResp: verifyResp,
	}
	go proposer.initVerifyPool()
	go proposer.run()
	listener, err := net.Listen("tcp", client)
	if err != nil {
		log.Fatal("net.Listen: ", err)
	}
	s := grpc.NewServer()
	clientpb.RegisterThemixServer(s, &server{reqc: reqc, repc: repc})
	err = s.Serve(listener)
	if err != nil {
		log.Fatal("s.Serve: ", err)
	}
}

func (proposer *Proposer) initVerifyPool() {
	for i := 0; i < runtime.NumCPU()-1; i++ {
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
			proposer.outputc <- &messagepb.Msg{}
		}
	}
}

func verify(req *clientpb.Payload) bool {
	// TODO(chenzx): To be implemented.
	return true
}
