FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init mock_clark && go build -o mock_clark .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mock_clark .
EXPOSE 8080
CMD ["./mock_clark"]
