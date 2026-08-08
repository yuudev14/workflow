.PHONY: seed-alerts drip-alerts storm-alerts proto
.PHONY: seed-alerts drip-alerts storm-alerts

rebuild-containers:
		docker compose -f ./docker/dev.docker-compose.yml up --build -d

rebuild-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build $(NAME) -d

rebuild-playbook-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build ytsoar_api -d

start-dev:
		docker compose -f ./docker/dev.docker-compose.yml up -d && \
		cd ./app/frontend && \
		npm run dev -- --hostname 0.0.0.0

start-frontend:
		cd ./app/frontend && \
		npm run dev -- --hostname 0.0.0.0

lint:
		cd ./app/frontend && \
		npm run lint

build:
		cd ./app/frontend && \
		npm run build

show-logs:
		docker compose -f ./docker/dev.docker-compose.yml logs --follow $(SERVICE)

# run the stack with the Go services under headless delve (VS Code: F5 attach,
# see .vscode/launch.json). Return to normal with make rebuild-containers / up.
start-debug:
		docker compose -f ./docker/dev.docker-compose.yml -f ./docker/debug.docker-compose.yml up -d && \
		cd ./app/frontend && \
		npm run dev -- --hostname 0.0.0.0

# Dev/demo alert data. Stdlib python on the HOST, talking to the dev stack over
# HTTP - see tools/alertgen/README.md. Not shipped, nothing imports it.
ALERTGEN := python3 ./tools/alertgen/alertgen.py

# ~40 alerts across all 5 source kinds and all severities, created_at spread
# over 14 days so the volume chart has a shape. Also escalates 3 into incidents.
seed-alerts:
		$(ALERTGEN) seed --count $(or $(N),40)

# One alert every ~20s until Ctrl-C. Leave running while clicking the UI.
drip-alerts:
		$(ALERTGEN) drip

# N byte-identical alerts. Must collapse to ONE row with dedup_count=N; if it
# does not, alerts_open_fingerprint_idx is wrong.
storm-alerts:
		$(ALERTGEN) storm --count $(or $(N),200)

# Install per-connector dependencies declared as <id>/requirements.txt (python,
# vendored into <id>/deps) or <id>/package.json (node, into <id>/node_modules).
# Runs INSIDE the api container: it has the RW mount of the tree and its
# alpine/musl wheels match what the sandbox executes.
connector-deps:
		docker compose -f ./docker/dev.docker-compose.yml exec ytsoar_api sh -c '\
		cd /opt/ytsoar/connectors && \
		for req in */requirements.txt; do \
			[ -e "$$req" ] || continue; \
			id=$${req%/requirements.txt}; \
			echo "==> pip: $$id"; \
			pip install --break-system-packages --no-cache-dir --quiet --target "$$id/deps" -r "$$req"; \
		done && \
		for pkg in */package.json; do \
			[ -e "$$pkg" ] || continue; \
			id=$${pkg%/package.json}; \
			echo "==> npm: $$id"; \
			npm install --prefix "$$id" --ignore-scripts --omit=dev --no-audit --no-fund; \
		done'
# protoc and the two plugins live in $(go env GOPATH)/bin. Output is
# gen/connectorruntimepb with paths=source_relative, which is what produced the
# committed files - regenerating any other way rewrites their package path.
proto:
		PATH="$$PATH:$$(go env GOPATH)/bin" protoc -I app/proto \
			--go_out=app/ytsoar/gen/connectorruntimepb --go_opt=paths=source_relative \
			--go-grpc_out=app/ytsoar/gen/connectorruntimepb --go-grpc_opt=paths=source_relative \
			connector_runtime.proto
