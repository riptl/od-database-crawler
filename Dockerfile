FROM golang:1.11.5 AS builder
ADD . /src
RUN cd /src \
 && go test . \
 && go build -o binary

FROM alpine
WORKDIR /app
COPY --from=builder /src/binary /app/
ENTRYPOINT ./binary
