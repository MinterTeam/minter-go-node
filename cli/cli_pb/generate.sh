#!/usr/bin/env bash

cd "$(dirname "$0")" || exit

protoc --go_out=. --go-grpc_out=. ./manager.proto