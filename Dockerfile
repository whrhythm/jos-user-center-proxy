FROM docker.1ms.run/library/golang:1.23 AS builder

COPY . /src
WORKDIR /src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -mod vendor -o main cmd/proxy/main.go

FROM docker.1ms.run/library/alpine:3.19

COPY --from=builder /src/main  /root/
RUN chmod +x /root/main

CMD ["/root/main"]