FROM golang:1.13-alpine

RUN apk add --no-cache git make gcc nodejs npm

ENV GO111MODULE=on
RUN go get github.com/mna/pigeon@7ee56e19b15c264d444de163f380df7c7c42a004
RUN go get golang.org/x/tools/cmd/goimports@b4207ef4930765bb7d8205504a8e9217f20be1dd
RUN npm install -g pegjs@0.10.0

