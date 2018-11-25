FROM golang:1.9

RUN add-apt-repository ppa:ethereum/ethereum && apt-get update && apt-get install solc

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine
RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080
CMD ["go-wrapper", "run"]
