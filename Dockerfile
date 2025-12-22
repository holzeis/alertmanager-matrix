# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app
ENV CGO_ENABLED=0

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o alertmanager-matrix

# ---- Runtime stage ----
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /app/alertmanager-matrix /alertmanager-matrix

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/alertmanager-matrix"]
