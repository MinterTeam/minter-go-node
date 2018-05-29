FROM ubuntu:12.04

# install git, wget and go
RUN apt-get update
RUN apt-get install -y git wget gcc
RUN wget https://dl.google.com/go/go1.10.linux-amd64.tar.gz
RUN tar -xvf go1.10.linux-amd64.tar.gz
RUN rm go1.10.linux-amd64.tar.gz
RUN mv go /usr/local
ENV GOROOT="/usr/local/go"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${GOROOT}/bin:${PATH}"

# install tendermint
RUN go get github.com/tendermint/tendermint/cmd/tendermint

# install Minter
WORKDIR /go/src/minter
ADD . .
RUN go get -d -v ./...
RUN go install -v ./...

# expose Minter Api port
EXPOSE 8841

# expose Tendermint port
EXPOSE 46657

CMD ["minter"]
