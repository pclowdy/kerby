#!/bin/bash
apt-get update && apt-get install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
export PATH=$PATH:$(go env GOPATH)/bin

if [ ! -f go.mod ]; then
    go mod init k8s-microservices-app
fi

go get google.golang.org/grpc@v1.60.1 google.golang.org/protobuf@v1.33.0

protoc --go_out=. --go_opt=module=k8s-microservices-app --go-grpc_out=. --go-grpc_opt=module=k8s-microservices-app proto/*.proto

go mod tidy
