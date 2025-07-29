FROM docker.1ms.run/library/golang:1.23 AS builder

ADD . /src
WORKDIR /src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -mod vendor -o main main.go initdb.go syncuser.go

FROM docker.1ms.run/library/alpine:latest

COPY --from=builder /src/main  /root/
RUN chmod +x /root/main

CMD ["/root/main"]