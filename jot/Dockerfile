FROM golang:1.24-alpine AS builder
WORKDIR /app/
COPY . .
RUN go build -o bin/app main.go

FROM alpine:3.19
WORKDIR /app/
COPY --from=builder /app/bin/app .
ENTRYPOINT ["./app"]
