FROM alpine:3.5

ADD builds/goruncmdsapi-linux-amd64.tar.gz /

EXPOSE 8080

CMD ["/goruncmdsapi"]