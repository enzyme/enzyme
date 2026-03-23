FROM golang:1.25.5 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o push-relay ./cmd/push-relay/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/push-relay /push-relay
ENTRYPOINT ["/push-relay"]
