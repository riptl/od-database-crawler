FROM golang:alpine as builder
ADD . /go/src/github.com/terorie/od-database-crawler
RUN apk add git && \
	CGO_ENABLED=0 go install \
		-d -v \
		-a -installsuffix cgo \
		-o /go/od-database-crawler \
		github.com/terorie/od-database-crawler

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/od-database-crawler /bin/
WORKDIR /oddb
VOLUME [ "/oddb" ]
CMD ["/bin/od-database-crawler", "server"]
