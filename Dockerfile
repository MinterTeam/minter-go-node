FROM tazhate/dockerfile-gox as builder

COPY . /gopath/src/github.com/MinterTeam/minter-go-node
WORKDIR /gopath/src/github.com/MinterTeam/minter-go-node
RUN apt-get update && apt-get install libleveldb-dev -y --no-install-recommends -q
RUN make get_tools
RUN make get_vendor_deps
RUN make build

FROM ubuntu:bionic

COPY --from=builder /gopath/src/github.com/MinterTeam/minter-go-node/build/minter/ /usr/bin/minter
RUN apt update && apt install libleveldb1v5 -y --no-install-recommends -q
RUN addgroup minteruser && useradd --no-log-init -r -m -d /minter -g minteruser minteruser
USER minteruser
WORKDIR /minter
EXPOSE 8841
ENTRYPOINT ["/usr/bin/minter"]
CMD ["node", "--home-dir", "/minter"]
