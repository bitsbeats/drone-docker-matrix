FROM golang:alpine AS builder

RUN true \
  && apk add -U --no-cache ca-certificates binutils

ADD . /go/src/github.com/bitsbeats/drone-docker-matrix
WORKDIR /go/src/github.com/bitsbeats/drone-docker-matrix

RUN true \
  && go build -mod=vendor . \
  && strip drone-docker-matrix

# ---

FROM docker

RUN true \
  && apk add -U --no-cache ca-certificates git

COPY --from=builder /go/src/github.com/bitsbeats/drone-docker-matrix/drone-docker-matrix /usr/local/bin/

CMD /usr/local/bin/drone-docker-matrix
