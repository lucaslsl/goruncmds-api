FROM alpine:3.5

ADD builds/goruncmds-linux-amd64.tar.gz /

EXPOSE 8080

CMD ["/goruncmds"]