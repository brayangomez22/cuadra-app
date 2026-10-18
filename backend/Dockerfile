# syntax=docker/dockerfile:1

FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# Migrations (embedded in the binary), for the end-to-end stack: `--target migrate`.
FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=build /out/migrate /migrate
USER nonroot:nonroot
ENTRYPOINT ["/migrate"]

# The API: the last stage, so it stays the default target.

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
