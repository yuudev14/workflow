from connectors.core.connector import Connector

BASE_URL = "https://ipinfo.io"


class IPinfo(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "lookup_ip":
            raise ValueError(f"operation ({operation}) does not exist in IPinfo")
        token = configs.get("token")
        return self.http(
            "GET",
            f"{configs.get('base_url') or BASE_URL}/{params['ip'].strip()}",
            headers={"Accept": "application/json", **({"Authorization": f"Bearer {token}"} if token else {})},
        )

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return self.execute(configs, {"ip": "8.8.8.8"}, "lookup_ip")
