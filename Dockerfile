FROM golang:1.23 AS builder

RUN mkdir /app
ADD main.go  /app/main.go
RUN go build -o /app/main /app/main.go

FROM alpine:latest
