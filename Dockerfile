FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/iced-mocha/reddit-client
COPY . /go/src/github.com/iced-mocha/reddit-client

RUN dep ensure -v && go install -v && /bin/bash -c "source workspace.env"

ENTRYPOINT ["reddit-client"]
