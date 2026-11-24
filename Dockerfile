# syntax=docker/dockerfile:1

# Build a static binary; the time zone database is embedded (time/tzdata).
FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The commit the image is built from, reported by /healthz and /readyz. CI
# passes it; .git stays out of the build context, so without it the service
# reports "unknown".
ARG COMMIT=
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X github.com/ruanpato/polite-greetings/internal/buildinfo.commit=${COMMIT}" \
      -o /out/server ./cmd/server

# Run it on an image with no shell or package manager, as a non-root user.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
ENV PORT=8080
EXPOSE 8080
# There is no curl in the image, so the binary checks itself against /readyz.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/server", "healthcheck"]
ENTRYPOINT ["/server"]
