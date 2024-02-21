FROM golang:1.20-alpine as builder

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
WORKDIR /build/reconciliation_lost_invoices
RUN go build -o main

# Start a new, final image to reduce size.
FROM alpine as final

# Copy the binaries and entrypoint from the builder image.
COPY --from=builder /build/reconciliation_lost_invoices/main /bin/

ENTRYPOINT [ "/bin/main" ]
