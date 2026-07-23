
FROM golang:1.26-alpine AS base
WORKDIR /moneymate

RUN apk add --no-cache git

ENV GONOSUMCHECK=* GONOSUMDB=*

COPY go.work go.work.sum ./

COPY shared/go.mod shared/go.mod
COPY services/auth/go.mod services/auth/go.mod
COPY services/gateway/go.mod services/gateway/go.mod
COPY services/merchant/go.mod services/merchant/go.mod

RUN go work sync

FROM base AS dev
WORKDIR /moneymate/services/auth

RUN go install github.com/air-verse/air@latest

CMD ["air", "-c", ".air.toml"]

FROM base AS builder
WORKDIR /moneymate

COPY shared/ shared/
COPY services/auth/ services/auth/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /bin/auth \
    ./services/auth/cmd/main.go

FROM alpine:3.19 AS prod

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=builder /bin/auth /bin/auth

EXPOSE 8081

CMD ["/bin/auth"]