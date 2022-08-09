#!/bin/sh
set -eu

peer=$1
dispatcher=$2

if [ ! -d log ]; then
  mkdir log
fi

name=$(printf 'peer%02d' "$peer")
go run cmd/peer/main.go --name "$name" --port $((peer + 3000)) --dispatcher "$dispatcher" >log/"${name}".log 2>&1
