FROM golang:1.13

RUN apt-get install -y curl

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine

RUN curl https://glide.sh/get | sh
RUN glide install

RUN go build -v -o ./bin/goldmine_api ./cmd/api
RUN go build -v -o ./bin/goldmine_consumer ./cmd/consumer
RUN go build -v -o ./bin/goldmine_migrate ./cmd/migrate
RUN ln -s ./bin/goldmine_api goldmine
RUN ln -s ./bin/goldmine_consumer goldmine_consumer
RUN ln -s ./bin/goldmine_migrate goldmine_migrate

EXPOSE 8080
ENTRYPOINT ["./goldmine"]
