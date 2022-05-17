# syntax = docker/dockerfile:1.2

FROM golang:1.17
WORKDIR /go/src/github.com/kris-nova/kush
ADD . .
RUN CGO_ENABLED=0 GOOS=linux make

FROM krisnova/kushbase:latest

# Copy the "root" directory as our "home" directory in the container
ADD root /root

# Install the kobfuscate binary
COPY --from=0 /go/src/github.com/kris-nova/kush/kobfuscate /bin/kobfuscate
WORKDIR /root
CMD ["/bin/kobfuscate", "-v"]
