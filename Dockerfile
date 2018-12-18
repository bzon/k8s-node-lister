FROM alpine:3.6

RUN apk add --no-cache ca-certificates tini

COPY ./bin/node-lister /node-lister

RUN chmod +x /node-lister

ENTRYPOINT ["/sbin/tini", "--", "/node-lister"]

