import base64

from connectors.core.connector import Connector

BASE_URL = "https://www.virustotal.com/api/v3"


def url_id(url: str) -> str:
    return base64.urlsafe_b64encode(url.encode()).decode().rstrip("=")


# operation: (param, path of the object)
OPERATIONS = {
    "ip_report": ("ip", lambda v: f"ip_addresses/{v}"),
    "domain_report": ("domain", lambda v: f"domains/{v}"),
    "url_report": ("url", lambda v: f"urls/{url_id(v)}"),
    "file_report": ("hash", lambda v: f"files/{v}"),
}


class VirusTotal(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation not in OPERATIONS:
            raise ValueError(f"operation ({operation}) does not exist in VirusTotal")
        param, path = OPERATIONS[operation]
        body = self.http(
            "GET",
            f"{configs.get('base_url') or BASE_URL}/{path(params[param].strip())}",
            headers={"x-apikey": configs.get("api_key", "")},
            missing_ok=True,
        )
        # an unknown object is a normal answer for a new file hash or url
        if body is None:
            return {"found": False, "malicious": 0, "suspicious": 0, "reputation": 0, "stats": {}}
        attributes = body["data"].get("attributes", {})
        stats = attributes.get("last_analysis_stats", {})
        return {
            "found": True,
            "malicious": stats.get("malicious", 0),
            "suspicious": stats.get("suspicious", 0),
            "reputation": attributes.get("reputation", 0),
            "stats": stats,
        }

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return self.execute(configs, {"ip": "8.8.8.8"}, "ip_report")
