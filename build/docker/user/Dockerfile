FROM golang:1.19.4-alpine3.17 AS builder

WORKDIR /build

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOPROXY https://goproxy.cn,direct

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

RUN go build  -ldflags="-s -w" -o user cmd/user/user.go

FROM alpine

WORKDIR /build

ENV TZ Asia/Shanghai

RUN mkdir -p configs/user
RUN mkdir -p logs

COPY --from=builder /build/user .
EXPOSE 8021
ENTRYPOINT ["./user"]
CMD ["-c", "configs/user/srv.yaml"]