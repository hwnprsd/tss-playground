build:
	@go build -o bin/tss

run:	build
	./bin/tss

proto: 	build
	@protoc --go_out=. --go-grpc_out=.  --grpc-gateway_out=logtostderr=true:. proto/*.proto
	@echo "protobuffs written"

node:	build
	./bin/tss node $(ARG)
