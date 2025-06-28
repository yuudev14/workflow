import logging
import sys
from colorlog import ColoredFormatter

from src.settings import settings


def setup_logging():
    # Create a ColoredFormatter
    formatter = ColoredFormatter(
        "%(log_color)s%(asctime)s - %(name)s - %(levelname)s - %(message)s%(reset)s",
        datefmt="%Y-%m-%d %H:%M:%S",
        reset=True,
        log_colors={
            'DEBUG': 'cyan',
            'INFO': 'green',
            'WARNING': 'yellow',
            'ERROR': 'red',
            'CRITICAL': 'red,bg_white',
        },
        secondary_log_colors={},
        style='%'
    )

    # Get the root logger
    logger = logging.getLogger(__name__)

    logger.setLevel(getattr(logging, settings.logging_level.upper()))

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)
    
    logger.addHandler(handler)

    return logger


logger = setup_logging()