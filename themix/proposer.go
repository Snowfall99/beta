package themix

type Proposer struct {
	reqc chan *Request
	repc chan *Response
}
