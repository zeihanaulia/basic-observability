FROM golang:alpine as golang
WORKDIR /go/src/order
COPY . .
# Static build required so that we can safely copy the binary over.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/order *.go; 

FROM alpine:latest as alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -q -r -0 /zoneinfo.zip .

FROM scratch
# the test program:
COPY --from=golang /go/src/order/bin/order /order
# the timezone data:
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /
# the tls certificates:
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/order"]
EXPOSE 3030