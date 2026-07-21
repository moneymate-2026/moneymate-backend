# ── Stage 1: base ───────────────────────────────────────────────
FROM golang:1.22-alpine AS base
WORKDIR /moneymate

# install dependencies
RUN apk add --no-cache git

# copy go workspace files first for layer caching
COPY go.work go.work.sum ./

# copy all go.mod files so workspace resolves
COPY shared/go.mod shared/go.mod
COPY services/auth/go.mod services/auth/go.mod
COPY services/gateway/go.mod services/gateway/go.mod
COPY services/merchant/go.mod services/merchant/go.mod

# download dependencies
RUN go work sync

# ── Stage 2: dev ────────────────────────────────────────────────
FROM base AS dev
WORKDIR /moneymate/services/merchant

# install air for hot reload
RUN go install github.com/air-verse/air@latest

# air watches for changes and rebuilds
CMD ["air", "-c", ".air.toml"]

# ── Stage 3: builder ────────────────────────────────────────────
FROM base AS builder
WORKDIR /moneymate

# copy all source needed for merchant
COPY shared/ shared/
COPY services/merchant/ services/merchant/

# build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /bin/merchant \
    ./services/merchant/cmd/main.go

# ── Stage 4: prod ───────────────────────────────────────────────
FROM alpine:3.19 AS prod

# security: don't run as root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=builder /bin/merchant /bin/merchant

# Expose Merchant gRPC port (Auth is 9091)
EXPOSE 9092

CMD ["/bin/merchant"]