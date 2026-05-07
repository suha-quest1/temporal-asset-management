# Multi-stage build for all Go services.
# Pass --build-arg SERVICE=worker|apiserver|demodriver|mockses|mockcreditfacility
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/service /bin/service
ENTRYPOINT ["/bin/service"]
