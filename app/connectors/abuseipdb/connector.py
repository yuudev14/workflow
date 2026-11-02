from connectors.core.connector import Connector

BASE_URL = "https://api.abuseipdb.com/api/v2"


class AbuseIPDB(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "check_ip":
            raise ValueError(f"operation ({operation}) does not exist in AbuseIPDB")
        body = self.http(
            "GET",
            f"{configs.get('base_url') or BASE_URL}/check",
            headers={"Key": configs.get("api_key", ""), "Accept": "application/json"},
            params={"ipAddress": params["ip"].strip(), "maxAgeInDays": params.get("max_age_days") or 90},
        )
        return body["data"]

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return self.execute(configs, {"ip": "127.0.0.1"}, "check_ip")
