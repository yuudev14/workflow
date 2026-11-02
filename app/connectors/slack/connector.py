from connectors.core.connector import Connector


class Slack(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "send_message":
            raise ValueError(f"operation ({operation}) does not exist in Slack")
        self.http("POST", configs["webhook_url"], json={"text": params["text"]})
        return {"sent": True}

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return {"configured": bool(configs.get("webhook_url"))}
