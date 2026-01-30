FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.25-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG GIT_TAG
ARG GIT_COMMIT

ENV CGO_ENABLED=0

# Install UPX for binary compression
RUN apk add --no-cache upx

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -ldflags="-s -w -X main.version=${GIT_TAG} -X main.commit=${GIT_COMMIT}" -o /usr/bin/singbox-checker . && \
  upx --best --lzma /usr/bin/singbox-checker

FROM alpine:3.21

LABEL org.opencontainers.image.source=https://github.com/knownasmobin/SingBox-checker

RUN apk add --no-cache ca-certificates curl tzdata && \
    adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /usr/bin/singbox-checker /usr/bin/singbox-checker

RUN mkdir -p /app/geo && \
    chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["/usr/bin/singbox-checker"]
