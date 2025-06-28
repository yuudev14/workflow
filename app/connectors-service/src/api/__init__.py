from fastapi import APIRouter
from src.api import celery
from src.api import connectors

routes = APIRouter()

routes.include_router(connectors.router, prefix="/connectors")
routes.include_router(celery.router)