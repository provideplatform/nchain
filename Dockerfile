FROM golang:1.13 AS builder

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine

RUN mkdir ~/.ssh && cp /go/src/github.com/provideapp/goldmine/ops/keys/ident-id_rsa ~/.ssh/id_rsa && chmod 0600 ~/.ssh/id_rsa && ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts
RUN git clone git@github.com:provideapp/ident.git /go/src/github.com/provideapp/ident && cd /go/src/github.com/provideapp/ident && git checkout master
RUN rm -rf ~/.ssh && rm -rf /go/src/github.com/provideapp/goldmine/ops/keys

WORKDIR /go/src/github.com/provideapp/goldmine
RUN make build

FROM golang:1.13

RUN mkdir -p /goldmine
WORKDIR /goldmine

COPY --from=builder /go/src/github.com/provideapp/goldmine/.bin /goldmine/.bin
COPY --from=builder /go/src/github.com/provideapp/goldmine/ops /goldmine/ops

EXPOSE 8080
ENTRYPOINT ["./ops/run_api.sh"]
