#!/usr/bin/env bash

cd "$(dirname "$0")" || exit

protoc --go_out=plugins=grpc:. ./manager.proto