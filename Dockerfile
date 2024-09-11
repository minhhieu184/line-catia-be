FROM golang:1.21-alpine as builder
RUN apk update && apk add openssh-client gcc g++ musl-dev git
WORKDIR /app
RUN go env -w GOPRIVATE=github.com/polkafoundry
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod go mod download
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build go build -o api cmd/api/*.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o bank cmd/bank/*.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o bot cmd/bot/*.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o migrate cmd/migrate/*.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o cron cmd/cron/*.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o leaderboard cmd/leaderboard/*.go

FROM alpine:latest
RUN apk add ca-certificates multirun
WORKDIR /app
COPY --from=builder /app/. ./
# CMD multirun "/app/api server" "/app/bot server" "/app/cron cron"
CMD ["/app/api", "server"]