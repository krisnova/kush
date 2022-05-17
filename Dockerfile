# syntax = docker/dockerfile:1.2

FROM golang:1.17
WORKDIR /go/src/github.com/kris-nova/kush
ADD . .
RUN CGO_ENABLED=0 GOOS=linux make

FROM alpine:latest
RUN apk add bash ncurses
RUN apk --no-cache add ca-certificates
ADD root /root
ADD root/hostname /etc/hostname
COPY --from=0 /go/src/github.com/kris-nova/kush/kush /bin/kush
WORKDIR /root
CMD ["/bin/kush"]
