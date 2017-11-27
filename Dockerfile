#
# [docker build]
# docker build -t git-http-transfer-build .
#
# [test]
# docker run --rm -v $PWD:/go/src/github.com/vvatanabe/go-git-http-transfer git-http-transfer-build bash -c "gotestcover -v -covermode=count -coverprofile=coverage.out ./..."
#
# [attach]
# docker run -it --rm -v $PWD:/go/src/github.com/vvatanabe/go-git-http-transfer -p 8080:8080 git-http-transfer-build bash
#
# [run server](in container)
# go run /go/src/github.com/vvatanabe/go-git-http-transfer/example/main.go -p 8080
# And
# gin -t ./example -p 8080 -a 5050
#
# OFFICIAL REPOSITORY: https://hub.docker.com/_/golang/
FROM golang:1.9

MAINTAINER Yuichi Watanabe

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
  && apt-get install -y --no-install-recommends vim curl wget unzip libssl-dev openssl ca-certificates git \
  && rm -fr /var/lib/apt/lists/*

RUN go get github.com/pierrre/gotestcover && go get github.com/codegangsta/gin

ENV GIT_ROOT_DIR /data/git
RUN mkdir -p $GIT_ROOT_DIR

ENV GIT_EXAMPLE_REPO_DIR /data/git/example.git
RUN mkdir -p $GIT_EXAMPLE_REPO_DIR && cd $GIT_EXAMPLE_REPO_DIR && git init --bare --shared

ENV GIT_TEST_REPO_DIR /data/git/test.git
RUN mkdir -p $GIT_TEST_REPO_DIR && cd $GIT_TEST_REPO_DIR && git init --bare --shared


ENV SRC_DIR /go/src/github.com/vvatanabe/go-git-http-transfer
RUN mkdir -p $SRC_DIR
WORKDIR $SRC_DIR

EXPOSE 8080