# Build Stage
FROM golang:1.9.2 AS build-env

ENV DEP_VERSION 0.3.2

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 && chmod +x /usr/local/bin/dep


ADD . /go/src/github.com/foomo/variant-balancer
WORKDIR /go/src/github.com/foomo/variant-balancer

RUN dep ensure -vendor-only
# install the dependencies without checking for go code

RUN go build -o /balancer simplevariantbalancer.go

# Package Stage
FROM alpine

COPY --from=build-env /balancer /balancer
ENTRYPOINT ./balancer
