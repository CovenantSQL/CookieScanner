# build stage
FROM golang:stretch AS builder
WORKDIR /go/src/github.com/CovenantSQL/CookieTester
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go install -ldflags '-linkmode external -extldflags -static'

# stage runner
FROM zenika/alpine-chrome:latest
WORKDIR /app
USER root
RUN apk --no-cache add ca-certificates tini
USER chrome
COPY --from=builder /go/bin/CookieTester /app/
ENTRYPOINT ["/sbin/tini", "--", "/app/CookieTester", "server"]
CMD []