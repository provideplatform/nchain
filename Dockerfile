FROM golang:1.11

RUN apt-get install -y curl

RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.9/solc-static-linux > /usr/local/bin/solc-v0.5.9 && chmod +x /usr/local/bin/solc-v0.5.9
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.8/solc-static-linux > /usr/local/bin/solc-v0.5.8 && chmod +x /usr/local/bin/solc-v0.5.8
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.7/solc-static-linux > /usr/local/bin/solc-v0.5.7 && chmod +x /usr/local/bin/solc-v0.5.7
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.6/solc-static-linux > /usr/local/bin/solc-v0.5.6 && chmod +x /usr/local/bin/solc-v0.5.6
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.5/solc-static-linux > /usr/local/bin/solc-v0.5.5 && chmod +x /usr/local/bin/solc-v0.5.5
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.4/solc-static-linux > /usr/local/bin/solc-v0.5.4 && chmod +x /usr/local/bin/solc-v0.5.4
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.3/solc-static-linux > /usr/local/bin/solc-v0.5.3 && chmod +x /usr/local/bin/solc-v0.5.3
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.2/solc-static-linux > /usr/local/bin/solc-v0.5.2 && chmod +x /usr/local/bin/solc-v0.5.2
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.1/solc-static-linux > /usr/local/bin/solc-v0.5.1 && chmod +x /usr/local/bin/solc-v0.5.1
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.0/solc-static-linux > /usr/local/bin/solc-v0.5.0 && chmod +x /usr/local/bin/solc-v0.5.0
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.4.25/solc-static-linux > /usr/local/bin/solc-v0.4.25 && chmod +x /usr/local/bin/solc-v0.4.25
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.4.24/solc-static-linux > /usr/local/bin/solc-v0.4.24 && chmod +x /usr/local/bin/solc-v0.4.24
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.4.23/solc-static-linux > /usr/local/bin/solc-v0.4.23 && chmod +x /usr/local/bin/solc-v0.4.23
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.4.22/solc-static-linux > /usr/local/bin/solc-v0.4.22 && chmod +x /usr/local/bin/solc-v0.4.22
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.4.21/solc-static-linux > /usr/local/bin/solc-v0.4.21 && chmod +x /usr/local/bin/solc-v0.4.21

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine

RUN curl https://glide.sh/get | sh
RUN glide install
RUN go build -v -o ./bin/goldmine_api ./cmd/api
RUN go build -v -o ./bin/goldmine_consumer ./cmd/consumer

EXPOSE 8080
CMD ["./bin/goldmine_api"]
