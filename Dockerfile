FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/icedmocha/reddit-client
COPY . /go/src/github.com/icedmocha/reddit-client

ENV REDDIT_SECRET test

RUN dep ensure -v && go install -v && /bin/bash -c "source workspace.env"

ENTRYPOINT ["reddit-client"]
