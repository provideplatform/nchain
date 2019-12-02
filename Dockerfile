FROM golang:1.13

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine

RUN make build

EXPOSE 8080
ENTRYPOINT ["make", "run_api"]
