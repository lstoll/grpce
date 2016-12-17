generate: helloproto/hello.pb.go

helloproto/hello.pb.go: vendor helloproto/hello.proto
	@echo "Generating"
	protoc --proto_path=.:vendor:"$$GOPATH"/src --go_out=plugins=grpc:. helloproto/hello.proto
