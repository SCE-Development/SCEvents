# build stage
FROM golang:1.25-alpine as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN go build -o ./scevents ./cmd/server

# production stage
FROM alpine:latest

WORKDIR /app

COPY --from=build /app/scevents ./

EXPOSE 8080

CMD ["./scevents"]