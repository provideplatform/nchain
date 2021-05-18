FROM golang:1.15 AS builder

RUN mkdir -p /go/src/github.com/provideplatform
ADD . /go/src/github.com/provideplatform/nchain

RUN mkdir ~/.ssh && cp /go/src/github.com/provideplatform/nchain/ops/keys/ident-id_rsa ~/.ssh/id_rsa && chmod 0600 ~/.ssh/id_rsa && ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts
RUN git clone git@github.com:provideplatform/ident.git /go/src/github.com/provideplatform/ident && cd /go/src/github.com/provideplatform/ident
RUN rm -rf ~/.ssh && rm -rf /go/src/github.com/provideplatform/nchain/ops/keys

WORKDIR /go/src/github.com/provideplatform/nchain
RUN make build

FROM alpine

RUN apk add --no-cache bash libc6-compat

RUN mkdir -p /nchain
WORKDIR /nchain

COPY --from=builder /go/src/github.com/provideplatform/nchain/.bin /nchain/.bin
COPY --from=builder /go/src/github.com/provideplatform/nchain/ops /nchain/ops

EXPOSE 8080
ENTRYPOINT ["./ops/run_api.sh"]
