# workflow

generate grpc code for golang
```
protoc \
  -I app/proto \
  --go_out=app/workflow_service/internal/grpc/workflows \
  --go_opt=paths=source_relative,Mworkflow.proto=app/workflow_service/internal/grpc \
  --go-grpc_out=app/workflow_service/internal/grpc/workflows \
  --go-grpc_opt=paths=source_relative,Mworkflow.proto=app/workflow_service/internal/grpc \
  workflow.proto


python3 -m grpc_tools.protoc \
  -I app/proto \
  --python_out=app/connectors-service/src/grpc/workflows \
  --pyi_out=app/connectors-service/src/grpc/workflows \
  --grpc_python_out=app/connectors-service/src/grpc/workflows \
  workflow.proto
```


generate mocks
```
mockgen -source=./app/workflow_service/internal/workflows/service.go -destination=./app/workflow_service/internal/workflows/mocks/service_mock.go
```


generate coverage tests
```
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```