FROM golang:1.23.0 as builder

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .

## build
FROM alpine:latest as ebrock

RUN apk add -U tzdata \
    && cp /usr/share/zoneinfo/Etc/UTC /etc/localtime \
    && apk del tzdata

WORKDIR /app

## set builder
COPY --from=builder /app/ebrock .

EXPOSE 8880

ENTRYPOINT ["/app/ebrock"]

