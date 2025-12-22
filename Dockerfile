# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Enable static binary
ENV CGO_ENABLED=0

# Copy source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN go build -o alertmanager-matrix

# ---- Runtime stage ----
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/alertmanager-matrix /alertmanager-matrix

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/alertmanager-matrix"]
