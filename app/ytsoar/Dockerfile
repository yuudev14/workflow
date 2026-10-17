# Prod images (dev: Dockerfile.dev). Build one with `docker build --target <stage>`.

# One build for the three binaries.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/worker ./cmd/sandbox

# distroless/static: CA certs (http_request connector) and a nonroot user.
FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /worker
ENTRYPOINT ["/worker"]

# Migrations run before start (#21: one-time job). pip/npm stay until #25.
FROM alpine:3.22 AS api
RUN apk add --no-cache python3 py3-pip nodejs-current npm \
    && adduser -D -u 10001 ytsoar
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY --from=build /out/api /usr/local/bin/api
COPY db/migrations /migrations
USER ytsoar
ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["sh", "-c", "goose -dir /migrations postgres \"postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable\" up && exec api"]

# Python and node runtimes, no Go toolchain.
FROM alpine:3.22 AS sandbox
RUN apk add --no-cache python3 py3-pip py3-jinja2 py3-requests nodejs-current npm \
    && pip install --break-system-packages --no-cache-dir colorlog pydantic-settings \
    && mkdir -p /opt/ytsoar/harness \
    && npm install --prefix /opt/ytsoar/harness nunjucks@3 \
    && apk del py3-pip npm \
    && adduser -D -u 10001 ytsoar
ENV NODE_PATH=/opt/ytsoar/harness/node_modules
COPY --from=build /out/sandbox /usr/local/bin/sandbox
USER ytsoar
EXPOSE 50052
ENTRYPOINT ["sandbox"]
