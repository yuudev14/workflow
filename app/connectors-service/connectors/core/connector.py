from abc import ABC, abstractmethod


from collections import ChainMap
import importlib
import inspect
import tomllib
from jinja2 import Template, Environment

from src.logger.logging import logger

env = Environment()


class Connector(ABC):

    @abstractmethod
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        raise NotImplementedError(
            f"execute function is not implemented in {self.__class__.__name__}"
        )

    @abstractmethod
    def health_check(
        self, configs: dict, params: dict, operation: str, *args, **kwargs
    ):
        raise NotImplementedError(
            f"execute function is not implemented in {self.__class__.__name__}"
        )

    @classmethod
    def consolidate_results(cls, *args: tuple[dict] | dict | list[dict]) -> dict:
        """
        This consolidates all the results

        Args:
            args (tuple[dict] | dict | list[dict]) = all results in the tasks

        Returns:
            consolidated results
        """
        results = dict()
        for arg in args:
            if isinstance(arg, dict):
                results = dict(ChainMap(results, arg))
            elif isinstance(arg, list) or isinstance(arg, tuple):
                results = dict(ChainMap(*arg))
        return results

    @classmethod
    def get_class_container(cls, connector_name: str) -> "Connector":
        """
        Get the class instance of the container

        Args:
            connector_name (str)

        Returns:
            the connector instance
        """
        logger.debug(f"getting the class connector for {connector_name}")
        module: Connector = importlib.import_module(
            f"connectors.{connector_name}.connector"
        )
        connector_classes = []
        for _, obj in inspect.getmembers(module):
            if inspect.isclass(obj) and issubclass(obj, Connector) and obj != Connector:
                connector_classes.append(obj)

        if len(connector_classes) > 1:
            logger.warning(
                f"found {len(connector_classes)} classes that inherits from Connector. Choosing the first class integrated"
            )

        connector: Connector = connector_classes[0]()
        logger.debug(f"class connector is {connector}")
        return connector

    @classmethod
    def get_connector_config(cls, config_name: str | None, connector_name: str) -> dict:
        """
        Get the connectors config

        Args:
            config_name (str) = configuration name
            connector_name (str) = connector name

        Returns:
            the configuration in dictionary. if config_name is None, return '{}'
        """
        logger.info(f"getting the config ({config_name}) for {connector_name}")
        if config_name is not None:
            with open(
                f"./connectors/{connector_name}/configs/{config_name}.toml", "rb"
            ) as f:
                return tomllib.load(f)
        logger.debug("No config available. return '{}'")
        return {}

    @classmethod
    def evaluate_params(cls, parameters: dict | None, variables: dict) -> dict:
        """
        Evaluate params using jinja

        Args:
            parameters (dict) = parameters of the task operation
            results (dict) = variables results to be rendered

        Returns:
            evaluated params where expected data is rendered

        """
        if parameters is None:
            return {}
        if not isinstance(parameters, dict):
            return Exception("parameter is not a value dict")
        for key, val in parameters.items():
            if isinstance(val, dict):
                parameters[key] = cls.evaluate_params(val, variables)
            elif isinstance(val, list):
                for i in range(len(val)):
                    parameters[key][i] = cls.evaluate_params(
                        parameters[key][i], variables
                    )
            elif isinstance(val, str):
                template = Template(val)
                print(variables)
                rendered_template = template.render(var=variables)
                parameters[key] = rendered_template
        return parameters
