build:
	@go build -o bin/tss

run:	build
	./bin/tss

proto: 	build
	@protoc --go_out=. --go-grpc_out=.  proto/*.proto
	@echo "protobuffs written"


