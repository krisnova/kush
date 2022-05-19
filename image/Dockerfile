FROM alpine:latest
RUN apk add bash ncurses curl nmap zmap wget strace emacs
RUN apk --no-cache add ca-certificates
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin
