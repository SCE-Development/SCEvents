FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
RUN go build -o ./scevents ./cmd/server

EXPOSE 8080

CMD ["./scevents"]