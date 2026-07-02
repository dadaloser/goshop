# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.22

FROM golang:${GO_VERSION}-alpine AS builder

ARG CMD_PATH
ARG GOPROXY=https://goproxy.cn,direct
ARG VERSION=dev
ARG VCS_REF=unknown

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN test -n "${CMD_PATH}" \
    && go build -trimpath -ldflags="-s -w" -o /out/goshop "${CMD_PATH}"

FROM alpine:${ALPINE_VERSION}

ARG SERVICE_NAME=goshop
ARG CONFIG_PATH
ARG GRPC_PORT
ARG HTTP_PORT
ARG VERSION=dev
ARG VCS_REF=unknown

ENV TZ=Asia/Shanghai \
    GIN_MODE=release \
    APP_CONFIG=/app/${CONFIG_PATH}

WORKDIR /app

LABEL org.opencontainers.image.title="${SERVICE_NAME}" \
      org.opencontainers.image.description="goshop service image built from a shared Dockerfile" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S goshop \
    && adduser -S -G goshop goshop \
    && mkdir -p /app/logs /app/configs \
    && chown -R goshop:goshop /app

COPY --from=builder /out/goshop /app/goshop
COPY --from=builder /src/${CONFIG_PATH} /app/${CONFIG_PATH}

RUN test -n "${CONFIG_PATH}" \
    && test -f "/app/${CONFIG_PATH}" \
    && test -n "${GRPC_PORT}" \
    && test -n "${HTTP_PORT}"

USER goshop

EXPOSE ${GRPC_PORT} ${HTTP_PORT}

ENTRYPOINT ["sh", "-c", "exec /app/goshop -c \"$APP_CONFIG\""]
