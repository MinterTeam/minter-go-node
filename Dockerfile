FROM tcnksm/gox as builder
COPY . /gopath/src/github.com/MinterTeam/minter-go-node
WORKDIR /gopath/src/github.com/MinterTeam/minter-go-node
RUN make get_tools
RUN make get_vendor_deps
RUN make buildc

FROM ubuntu
COPY --from=builder /gopath/src/github.com/MinterTeam/minter-go-node/build/minter/ /usr/bin/minter
ENV MINTERHOME /minter
RUN apt update && apt install libleveldb1v5 libleveldb-dev -y 
RUN addgroup minteruser && \
    useradd --no-log-init -r -g minteruser minteruser -d "$MINTERHOME"

USER minteruser
VOLUME [ $MINTERHOME ]
WORKDIR $MINTERHOME
EXPOSE 8841
ENTRYPOINT ["/usr/bin/minter"]
