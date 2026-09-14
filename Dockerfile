# Build stage
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o server .

# Final runtime stage
FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/server /server
CMD ["/server"]
