FROM golang:1.23-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/dw0rdwk-api ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /out/dw0rdwk-api /app/dw0rdwk-api
EXPOSE 8080
ENTRYPOINT ["/app/dw0rdwk-api"]
