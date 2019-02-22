FROM golang:alpine as builder
RUN apk add git && \
	go get -d -v github.com/terorie/od-database-crawler
WORKDIR /go/src/github.com/terorie/od-database-crawler
RUN	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/od-database-crawler .

FROM scratch
COPY script/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/od-database-crawler /go/
VOLUME [ "/go" ]
CMD ["/go/od-database-crawler", "server"]