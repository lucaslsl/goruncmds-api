FROM alpine:3.5

ADD builds/goruncmds-0.1.1-linux-amd64.tar.gz /

EXPOSE 8080

CMD ["/goruncmds"]