# syntax=docker/dockerfile:1

# Build a static binary; the time zone database is embedded (time/tzdata).
FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Run it on an image with no shell or package manager, as a non-root user.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/server"]
