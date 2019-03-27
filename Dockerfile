FROM golang:1.9

RUN apt-get install -y curl
RUN curl -L https://github.com/ethereum/solidity/releases/download/v0.5.3/solc-static-linux > /usr/local/bin/solc-v0.5.4 && chmod +x /usr/local/bin/solc-v0.5.4
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
RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080
CMD ["go-wrapper", "run"]
