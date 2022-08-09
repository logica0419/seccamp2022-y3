FROM golang:1.19.0-alpine
RUN apk update && mkdir /go/src/app
WORKDIR /go/src/app
