package themix

type Node struct {
	inputc    chan *Msg
	outputc   chan *Msg
	reqc      chan *Request
	repc      chan *Response
	themixQue ThemixQue
}
