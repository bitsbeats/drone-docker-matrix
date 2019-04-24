FROM golang:alpine AS builder

RUN true \
  && apk add -U --no-cache ca-certificates git binutils

ADD . /go/src/github.com/bitsbeats/drone-docker-matrix
WORKDIR /go/src/github.com/bitsbeats/drone-docker-matrix

ENV CGO_ENABLED=0

RUN true \
  && go get github.com/golang/dep/cmd/dep \
  && dep ensure -v \
  && go test . -test.count 1000 \
  && go build . \
  && strip drone-docker-matrix

# ---

FROM docker

RUN true \
  && apk add -U --no-cache ca-certificates git

COPY --from=builder /go/src/github.com/bitsbeats/drone-docker-matrix/drone-docker-matrix /usr/local/bin/

CMD /usr/local/bin/drone-docker-matrix
