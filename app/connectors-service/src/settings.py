from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    app_name: str = "Awesome API"
    logging_level: str
    celery_broker: str = "pyamqp://guest:guest@localhost:5672"
    celery_backend: str = "db+postgresql://postgres:password@localhost:5432/celery_logs?sslmode=disable"

    mq_url: str = "amqp://guest:guest@localhost:5672/"
    workflow_queue: str = "workflow"
    workflow_processor_queue: str = "workflow_processor"

    model_config = SettingsConfigDict(env_file=".env")


settings = Settings()
