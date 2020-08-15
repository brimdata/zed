FROM golang:1.14-alpine AS build

ARG LDFLAGS

# All these steps will be cached
RUN mkdir /build
WORKDIR /build

# Copying the .mod and .sum files before the rest of the code
# to improve the caching behavior of Docker
COPY go.mod go.sum . 

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

# And compile the project 
# CGO_ENABLED and installsuffix are part of the scheme to get better caching on builds
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -a -installsuffix cgo -o /go/bin/zqd ./cmd/zqd


FROM golang:1.14-alpine
WORKDIR /app
RUN apk update && apk add ca-certificates
COPY --from=build /go/bin/zqd /go/bin
