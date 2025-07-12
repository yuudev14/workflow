from fastapi import APIRouter, Path as FastApiPath, HTTPException
from src.core.workflow import WorkflowGraph
from src.logger.logging import logger
import json
from pathlib import Path


router = APIRouter()
CONNECTORS_DIR = "./connectors"

class ConnectorsController:
    def __init__(self):
        pass


    @router.get("/")
    async def get_all_connectors():
        path = Path(CONNECTORS_DIR)
        connectors_data = []
        for item in path.iterdir():
            if item.name != "core":
                connectors_path = Path(CONNECTORS_DIR + "/" + item.name + "/info.json")
                try:
                    with connectors_path.open() as file:
                        connector_info = json.load(file)
                        # check for configs
                        config_path = Path(CONNECTORS_DIR + "/" + item.name + "/configs")
                        if config_path.exists():
                            configs = [config.stem for config in config_path.iterdir() if config.is_file()]
                            print(configs)
                            connector_info["configs"] = configs
                        connectors_data.append(connector_info)
                except json.JSONDecodeError:
                    logger.warning(f"Error decoding JSON from {connectors_path}")
                except FileNotFoundError:
                    logger.warning(f"File not found: {connectors_path}")
                except Exception as e:
                    logger.warning(f"An error occurred while processing {connectors_path}: {str(e)}")



        return connectors_data
    
    @router.get("/{connector_id}")
    async def get_connector_info(
        connector_id: str = FastApiPath(..., description="connector name")
    ):
        try:
            connectors_path = Path(CONNECTORS_DIR + "/" + connector_id + "/info.json")
            with connectors_path.open() as file:
                return json.load(file)
        except FileNotFoundError as e:
            raise HTTPException(status_code=404, detail=str(e))
