FROM golang:1.24.4-alpine AS builder

EXPOSE 8080

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o clock cmd/main.go

FROM alpine:latest

WORKDIR /app

ENV TZ=Europe/Paris
RUN apk --no-cache add tzdata

COPY --from=builder /app/clock /app/clock
COPY --from=builder /app/static /app/static


CMD ["./clock"]