# Each compose service picks a stage with `target:`. Docker builds a shared
# stage once, so `docker compose up --build` runs the base work one time.
#
#   dev stages:  dev-base -> dev-worker | dev-api | dev-sandbox -> dev -> debug
#   prod stages: build -> api | worker | sandbox

# ---------- dev ----------

# Go toolchain, air and the module cache. No `COPY . .`: in dev the bind mount
# ../app/ytsoar:/app gives the source.
FROM golang:1.26-alpine AS dev-base
RUN apk add --no-cache make
RUN go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
ENV GIN_MODE=debug AIR_POLL=true
EXPOSE 8080

# The worker runs no user code: it needs only Go.
FROM dev-base AS dev-worker

# The api runs the migrations, and pip/npm install connector dependencies
# after an upload (internal/adapters/depsinstall; #25 moves this).
FROM dev-base AS dev-api
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
RUN apk add --no-cache python3 py3-pip nodejs-current npm
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]

# The sandbox runs python and node code. nodejs-current (24.x), NOT nodejs
# (22.x): the node harnesses and .ts connectors rely on native TypeScript
# type stripping (Node >= 23.6).
FROM dev-base AS dev-sandbox
RUN apk add --no-cache python3 py3-pip py3-jinja2 py3-requests nodejs-current npm \
    && pip install --break-system-packages --no-cache-dir colorlog pydantic-settings \
    && mkdir -p /opt/ytsoar/harness \
    && npm install --prefix /opt/ytsoar/harness nunjucks@3
ENV NODE_PATH=/opt/ytsoar/harness/node_modules

# One process (cmd/dev): api + worker + sandbox. It needs the sandbox
# runtimes and goose. pip and npm come with the sandbox stage.
FROM dev-sandbox AS dev
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]

# Only docker/debug.docker-compose.yml uses this.
FROM dev AS debug
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# ---------- prod ----------

# One go build for the three static binaries.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/worker ./cmd/sandbox

# The worker: the binary only. distroless/static has CA certs (for the
# http_request connector) and a nonroot user.
FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /worker
ENTRYPOINT ["/worker"]

# The api: goose and the migrations run before it starts. pip/npm stay until
# #25 moves the connector dependency install. #21 can replace the migration
# step with a one-time job.
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

# The sandbox: the python and node runtimes, no Go toolchain.
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
