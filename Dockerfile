FROM alpine:3.5

ADD builds/goruncmds-0.1.3-linux-amd64.tar.gz /

EXPOSE 8080

CMD ["/goruncmds"]