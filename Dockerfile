FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/icedmocha/reddit
COPY . /go/src/github.com/icedmocha/reddit

RUN dep ensure && go install

ENTRYPOINT ["reddit"]
