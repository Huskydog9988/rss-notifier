# syntax=docker/dockerfile:1

# build
FROM golang:1.18-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /rss-notifier

# deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build rss-notifier rss-notifier

# USER nonroot:nonroot

CMD ["/rss-notifier"]

# ENTRYPOINT [ "/rss-notifier" ]