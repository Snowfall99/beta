all:
	protoc --go_out=. --go-grpc_out=. *.proto
	go build -o client main.go 
