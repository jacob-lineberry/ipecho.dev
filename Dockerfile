# syntax=docker/dockerfile:1

# ========================================
# build stage: compile static go binary
# ========================================
# base images are pinned by digest; dependabot bumps the tag and digest together.
# the golang image sets GOTOOLCHAIN=local, so a go.mod newer than this image fails the build.
FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build

WORKDIR /src

# copy deps
COPY go.mod go.sum ./

# download deps with buildkit cache mount
# GHA will cachce /go/pkg/mod across builds
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

# copy src only
COPY cmd ./cmd
COPY internal ./internal

# build args for cross-platform compilation and version stamping
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

# build static binary with optimizations
# cache mount for go-build speeds up repeated builds
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 \
  GOOS=${TARGETOS:-linux} \
  GOARCH=${TARGETARCH:-amd64} \
  go build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o /out/ipecho-api \
  ./cmd/server

# ========================================
# runtime stage: minimal distroless image
# ========================================
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

# OCI labels for supply chain tracking
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="ipecho-api" \
  org.opencontainers.image.description="Minimal IP echo API" \
  org.opencontainers.image.vendor="Code After Dark" \
  org.opencontainers.image.source="https://github.com/jacob-lineberry/ipecho.dev" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${VCS_REF}" \
  org.opencontainers.image.created="${BUILD_DATE}" \
  org.opencontainers.image.licenses="MIT"

# copy only the compiled binary from build stage
COPY --from=build /out/ipecho-api /ipecho-api

# document port (GCR will set the $PORT env var)
EXPOSE 8080

# run as non-root user (distroless default: 65532:65532)
USER nonroot:nonroot

# execute the binary
ENTRYPOINT ["/ipecho-api"]

