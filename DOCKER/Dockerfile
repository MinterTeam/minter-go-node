FROM ubuntu:12.04

RUN apt-get update

# api port
EXPOSE 8841 46658

ENTRYPOINT ["/usr/bin/minter"]
CMD ["/usr/bin/minter"]
STOPSIGNAL SIGTERM

ARG BINARY=minter
COPY $BINARY /usr/bin/minter

