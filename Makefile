
rebuild-containers:
		docker compose -f ./docker/dev.docker-compose.yml up --build -d

rebuild-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build $(NAME) -d

rebuild-playbook-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build ytsoar_api -d

start-dev:
		docker compose -f ./docker/dev.docker-compose.yml up -d && \
		cd ./app/frontend && \
		npm run dev

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
		docker compose -f ./docker/dev.docker-compose.yml -f ./docker/debug.docker-compose.yml up -d

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