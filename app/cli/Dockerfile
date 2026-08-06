# Dockerfile for resma-cli (CLI only)
# Unlike the API, the CLI is pure Go and does not require DuckDB/CGO.
# This allows us to use a static distroless runtime image.

########################################
# Stage 1: Builder
########################################
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# Download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a static binary (no CGO needed)
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/resma-swarm/resma/app/cli/internal/version.Version=dev" \
    -o /out/resma \
    ./cmd/resma-cli

########################################
# Stage 2: Runtime
########################################
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/resma /resma

ENTRYPOINT ["/resma"]
