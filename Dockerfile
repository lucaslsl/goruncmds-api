FROM alpine:3.5

ADD goruncmdsapi /

EXPOSE 8080

CMD ["/goruncmdsapi"]