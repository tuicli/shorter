FROM golang:1.24-alpine AS base

WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM base AS build-backend
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend

FROM base AS build-bot
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM base AS build-migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22 AS backend
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-backend /out/backend /app/backend
USER 65532:65532
ENTRYPOINT ["/app/backend"]

FROM alpine:3.22 AS bot
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-bot /out/bot /app/bot
USER 65532:65532
ENTRYPOINT ["/app/bot"]

FROM alpine:3.22 AS migrate
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-migrate /out/migrate /app/migrate
COPY migrations /app/migrations
USER 65532:65532
ENTRYPOINT ["/app/migrate"]
