
rebuild-containers:
		docker compose -f ./docker/dev.docker-compose.yml up --build -d

rebuild-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build $(NAME) -d

rebuild-workflow-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build workflow_service -d

start-dev:
		docker compose -f ./docker/dev.docker-compose.yml up -d

show-logs:
		docker compose -f ./docker/dev.docker-compose.yml logs --follow workflow_worker connector_service workflow_service frontend