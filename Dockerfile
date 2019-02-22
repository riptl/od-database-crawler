FROM golang:alpine as builder
ADD . /go/src/github.com/terorie/od-database-crawler
RUN apk add git \
 && go get -d -v github.com/terorie/od-database-crawler \
 && CGO_ENABLED=0 go install -a \
    -installsuffix cgo \
    -ldflags="-s -w" \
    github.com/terorie/od-database-crawler

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/od-database-crawler /bin/
WORKDIR /oddb
VOLUME [ "/oddb" ]
CMD ["/bin/od-database-crawler", "server"]
