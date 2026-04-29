FROM golang:1.25.3-bookworm AS builder

WORKDIR /app

# Copy dependency manifests first (cache friendly)
COPY shared/go.mod shared/go.sum ./shared/
COPY pb/go.mod pb/go.sum ./pb/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY service/withdraw/go.mod service/withdraw/go.sum ./service/withdraw/

WORKDIR /app/service/withdraw

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

WORKDIR /app

# Copy source
COPY shared/ ./shared/
COPY pb/ ./pb/
COPY pkg/ ./pkg/
COPY service/withdraw/ ./service/withdraw/

WORKDIR /app/service/withdraw

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/withdraw_bin ./cmd/main.go


FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 appuser && \
    adduser -D -s /bin/sh -u 1000 -G appuser appuser

WORKDIR /app

RUN mkdir -p /var/log/app && chown -R 1000:1000 /var/log/app

COPY --from=builder /app/withdraw_bin ./withdraw
COPY service/withdraw/migrations ./migrations

RUN chown -R 1000:1000 ./withdraw ./migrations

USER appuser

EXPOSE 50060

CMD ["./withdraw"]
