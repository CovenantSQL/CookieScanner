# build stage
FROM golang:alphine AS builder
WORKDIR /go/src/github.com/CovenantSQL/CookieTester
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go install -ldflags '-linkmode external -extldflags -static'

# stage runner
FROM browserless/chrome:1.5.0-chrome-stable
WORKDIR /app
COPY --from=builder /go/bin/CookieTester /app/
ENTRYPOINT ./CookieTester
CMD []