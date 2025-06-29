
rebuild-containers:
		docker compose -f ./docker/dev.docker-compose.yml up --build -d

rebuild-service:
		docker compose -f ./docker/dev.docker-compose.yml up --build $(NAME) -d

start-dev:
		docker compose -f ./docker/dev.docker-compose.yml up
