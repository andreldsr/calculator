# syntax=docker/dockerfile:1

# Build: a static binary, so the runtime image needs no libc.
FROM golang:1.27 AS build
WORKDIR /src
# Standard library only: go.mod is the whole dependency graph.
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Tests with the race detector, which needs cgo. Not part of the default build:
#   docker build --target test backend
FROM build AS test
RUN go vet ./... && go test -race -cover ./...

# Runtime: no shell or package manager, runs as an unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
# The image has no curl or wget, so the binary checks itself.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/api", "-healthcheck"]
ENTRYPOINT ["/api"]
