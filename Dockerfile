# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG GIT_TAG
ARG GIT_COMMIT
ARG USERNAME=kutovoys
ARG REPOSITORY_NAME=singbox-checker

ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/github.com/${USERNAME}/${REPOSITORY_NAME}

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -ldflags="-X main.version=${GIT_TAG} -X main.commit=${GIT_COMMIT}" -a -installsuffix cgo -o /usr/bin/singbox-checker .

# Download sing-box
FROM --platform=${BUILDPLATFORM:-linux/amd64} alpine:latest AS singbox
RUN apk add --no-cache curl
ARG TARGETARCH
RUN case "${TARGETARCH}" in \
      "amd64") ARCH="amd64" ;; \
      "arm64") ARCH="arm64" ;; \
      *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -L -o /sing-box.tgz https://github.com/SagerNet/sing-box/releases/latest/download/sing-box-1.8.0-linux-${ARCH}.tar.gz && \
    tar -xzf /sing-box.tgz -C / && \
    mv /sing-box-*/sing-box /usr/local/bin/sing-box && \
    chmod +x /usr/local/bin/sing-box

# Final stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source=https://github.com/${USERNAME}/${REPOSITORY_NAME}

WORKDIR /app
COPY --from=builder /usr/bin/singbox-checker /
COPY --from=singbox /usr/local/bin/sing-box /usr/local/bin/sing-box
USER nonroot:nonroot

ENTRYPOINT ["/singbox-checker"]