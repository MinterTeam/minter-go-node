#!/usr/bin/env bash

cd "$(dirname "$0")" || exit

protoc -I/usr/local/include -I. \
    -I"$GOPATH"/src \
    -I"$GOPATH"/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --go_out=plugins=grpc:. ./api.proto

protoc -I/usr/local/include -I. \
    -I"$GOPATH"/src \
    -I"$GOPATH"/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --grpc-gateway_out=logtostderr=true:. ./api.proto

protoc -I/usr/local/include -I. \
    -I"$GOPATH"/src \
    -I"$GOPATH"/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --swagger_out=logtostderr=true:. ./api.proto