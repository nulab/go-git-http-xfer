# OFFICIAL REPOSITORY: https://hub.docker.com/_/golang/
FROM golang:1.10.0

MAINTAINER Yuichi Watanabe

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
  && apt-get install -y --no-install-recommends vim curl wget unzip libssl-dev openssl ca-certificates git \
  && rm -fr /var/lib/apt/lists/*

RUN go get github.com/codegangsta/gin

ENV GIT_ROOT_DIR /data/git
RUN mkdir -p $GIT_ROOT_DIR

ENV GIT_EXAMPLE_REPO_DIR /data/git/example.git
RUN mkdir -p $GIT_EXAMPLE_REPO_DIR && cd $GIT_EXAMPLE_REPO_DIR && git init --bare --shared

ENV GIT_TEST_REPO_DIR /data/git/test.git
RUN mkdir -p $GIT_TEST_REPO_DIR && cd $GIT_TEST_REPO_DIR && git init --bare --shared


ENV SRC_DIR /go/src/github.com/nulab/go-git-http-xfer
RUN mkdir -p $SRC_DIR
WORKDIR $SRC_DIR

EXPOSE 5050