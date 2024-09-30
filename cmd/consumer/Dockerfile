FROM golang:1.23.1
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o ./cmd/consumer ./cmd/consumer
CMD ["./cmd/consumer/consumer"]