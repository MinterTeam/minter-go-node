FROM ubuntu:18.04

RUN apt-get update && apt-get install -y software-properties-common build-essential wget
RUN apt-get install -y libsnappy-dev
RUN wget https://github.com/google/leveldb/archive/v1.20.tar.gz && \
      tar -zxvf v1.20.tar.gz && \
      cd leveldb-1.20/ && \
      make && \
      cp -r out-static/lib* out-shared/lib* /usr/local/lib/ && \
      cd include/ && \
      cp -r leveldb /usr/local/include/ && \
      ldconfig && \
      rm -f v1.20.tar.gz
RUN wget https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.12.1.linux-amd64.tar.gz

ENV GOPATH=$HOME/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

WORKDIR /go/src/github.com/MinterTeam/minter-go-node
