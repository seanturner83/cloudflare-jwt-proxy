FROM golang as builder
WORKDIR /go/src/app
ADD main.go .
RUN go get .
RUN go build -ldflags "-linkmode external -extldflags -static" -a main.go

FROM alpine as alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /zoneinfo.zip .
RUN adduser -D golang

FROM scratch
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /
COPY --from=alpine /etc/ssl/certs /etc/ssl/certs
COPY --from=alpine /etc/passwd /etc/passwd
COPY --from=builder /go/src/app/main /cloudflare-jwt-proxy
USER golang
ENTRYPOINT ["/cloudflare-jwt-proxy"]
