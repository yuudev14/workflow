
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