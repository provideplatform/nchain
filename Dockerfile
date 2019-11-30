FROM golang:1.13

RUN mkdir -p /go/src/github.com/provideapp
ADD . /go/src/github.com/provideapp/goldmine
WORKDIR /go/src/github.com/provideapp/goldmine

RUN make build
RUN ln -s ./.bin/goldmine_api api
RUN ln -s ./.bin/goldmine_consumer consumer
RUN ln -s ./.bin/goldmine_migrate migrate
RUN ln -s ./.bin/goldmine_statsdaemon statsdaemon

EXPOSE 8080
ENTRYPOINT ["./api"]
