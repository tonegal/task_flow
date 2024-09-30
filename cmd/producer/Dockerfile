FROM golang:1.23.1
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o ./cmd/producer ./cmd/producer
CMD ["./cmd/producer/producer"]