
rebuild-containers:
		docker compose -f ./docker/dev.docker-compose.yml up --build -d

rebuild-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build $(NAME) -d

rebuild-workflow-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build workflow_service -d

start-dev:
		docker compose -f ./docker/dev.docker-compose.yml up -d && \
		cd ./app/workflow-frontend && \
		npm run dev

lint:
		cd ./app/workflow-frontend && \
		npm run lint

build:
		cd ./app/workflow-frontend && \
		npm run build

show-logs:
		docker compose -f ./docker/dev.docker-compose.yml logs --follow $(SERVICE)