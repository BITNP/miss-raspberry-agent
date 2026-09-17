# syntax=docker/dockerfile:1

# Build the static server binary.
FROM golang:1.26-bookworm AS build
WORKDIR /src

# Cache module downloads independently of the application source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/agent ./cmd/server

# The binary is static, so a distroless runtime is enough. It ships CA
# certificates (for HTTPS to the model API) and tzdata (for the time tool)
# without any shell or package manager.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/agent /agent
EXPOSE 4514
USER nonroot:nonroot
ENTRYPOINT ["/agent"]
