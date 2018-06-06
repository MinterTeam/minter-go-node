FROM alpine:3.7

ENV MINTERHOME /minter

RUN apk update && \
    apk upgrade && \
    apk --no-cache add curl jq bash && \
    addgroup minteruser && \
    adduser -S -G minteruser minteruser -h "$MINTERHOME"

USER minteruser

VOLUME [ $MINTERHOME ]

WORKDIR $MINTERHOME

# api port
EXPOSE 8841 46658

ENTRYPOINT ["/usr/bin/minter"]
CMD ["/usr/bin/minter"]
STOPSIGNAL SIGTERM

ARG BINARY=minter
COPY $BINARY /usr/bin/minter

