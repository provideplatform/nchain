FROM golang:1.15 AS builder

RUN mkdir -p /go/src/github.com/provideplatform
ADD . /go/src/github.com/provideplatform/nchain

WORKDIR /go/src/github.com/provideplatform/nchain
RUN make build

FROM alpine

RUN apk add --no-cache bash curl libc6-compat

RUN mkdir -p /nchain
WORKDIR /nchain

COPY --from=builder /go/src/github.com/provideplatform/nchain/.bin /nchain/.bin
COPY --from=builder /go/src/github.com/provideplatform/nchain/ops /nchain/ops

EXPOSE 8080
ENTRYPOINT ["./ops/run_api.sh"]
