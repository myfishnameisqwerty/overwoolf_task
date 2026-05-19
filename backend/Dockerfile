FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /backend .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /backend /app/backend
EXPOSE 8080
ENTRYPOINT ["/app/backend"]
