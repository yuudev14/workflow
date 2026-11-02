from connectors.core.connector import Connector


class Teams(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "send_message":
            raise ValueError(f"operation ({operation}) does not exist in Teams")
        body = [{"type": "TextBlock", "text": params["text"], "wrap": True}]
        if params.get("title"):
            body.insert(0, {"type": "TextBlock", "text": params["title"], "weight": "Bolder", "size": "Medium"})
        # the Workflows webhook of Teams takes an Adaptive Card in a message
        self.http("POST", configs["webhook_url"], json={
            "type": "message",
            "attachments": [{
                "contentType": "application/vnd.microsoft.card.adaptive",
                "content": {"type": "AdaptiveCard", "version": "1.4", "body": body},
            }],
        })
        return {"sent": True}

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return {"configured": bool(configs.get("webhook_url"))}
