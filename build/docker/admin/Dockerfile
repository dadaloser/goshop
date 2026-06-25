 FROM golang:1.19.4-alpine3.17 AS builder

WORKDIR /build

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOPROXY https://goproxy.cn,direct

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

RUN go build  -ldflags="-s -w" -o admin cmd/admin/admin.go

FROM alpine

WORKDIR /build

ENV TZ Asia/Shanghai

RUN mkdir -p configs/admin
RUN mkdir -p logs

COPY --from=builder /build/admin .
EXPOSE 8050
ENTRYPOINT ["./admin"]
CMD ["-c", "configs/admin/admin.yaml"]