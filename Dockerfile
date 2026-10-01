# ---- Build stage ----
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cache module downloads — only re-runs when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary.
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /profilepulse ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && \
    adduser -D -u 1001 appuser

COPY --from=build /profilepulse /usr/local/bin/profilepulse

USER appuser
EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=5s --retries=5 --start-period=5s \
  CMD wget -q -O - http://localhost:3000/health || exit 1

ENTRYPOINT ["profilepulse"]
