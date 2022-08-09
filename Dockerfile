FROM golang:1.18.2-alpine
RUN apk update && mkdir /go/src/app
WORKDIR /go/src/app
