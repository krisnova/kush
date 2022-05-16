# syntax = docker/dockerfile:1.2

FROM golang:1.17
WORKDIR /go/src/github.com/kris-nova/kush
COPY . .
COPY vendor vendor
RUN CGO_ENABLED=0 GOOS=linux make

FROM alpine:latest
RUN apk add nmap nmap-scripts
RUN apk --no-cache add ca-certificates
COPY root /root
COPY --from=0 /go/src/github.com/kris-nova/kush/kush /kush
WORKDIR /root
CMD ["/kush"]
