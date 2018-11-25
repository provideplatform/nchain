FROM golang:1.9

RUN apt-get install -y curl
RUN curl https://github.com/ethereum/solidity/releases/download/v0.5.0/solc-static-linux > /usr/local/bin/solc-v0.5.0 && chmod +x /usr/local/bin/solc-v0.5.0
RUN curl https://github.com/ethereum/solidity/releases/download/v0.4.25/solc-static-linux > /usr/local/bin/solc-v0.4.25 && chmod +x /usr/local/bin/solc-v0.4.25
RUN curl https://github.com/ethereum/solidity/releases/download/v0.4.24/solc-static-linux > /usr/local/bin/solc-v0.4.24 && chmod +x /usr/local/bin/solc-v0.4.24

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine
RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080
CMD ["go-wrapper", "run"]
