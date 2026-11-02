from datetime import datetime, timezone

from connectors.core.connector import Connector

BASE_URL = "https://rdap.org"


def event_date(body: dict, action: str):
    for event in body.get("events", []):
        if event.get("eventAction") == action:
            return event.get("eventDate")
    return None


def registrar(body: dict):
    for entity in body.get("entities", []):
        if "registrar" in entity.get("roles", []):
            for field in (entity.get("vcardArray") or [None, []])[1]:
                if field[0] == "fn":
                    return field[3]
    return None


class RDAP(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        if operation != "domain_lookup":
            raise ValueError(f"operation ({operation}) does not exist in RDAP")
        domain = params["domain"].strip().lower()
        body = self.http("GET", f"{configs.get('base_url') or BASE_URL}/domain/{domain}",
                         headers={"Accept": "application/rdap+json"})
        created = event_date(body, "registration")
        age_days = None
        if created:
            age_days = (datetime.now(timezone.utc) - datetime.fromisoformat(created.replace("Z", "+00:00"))).days
        return {
            "domain": domain,
            "registrar": registrar(body),
            "created": created,
            "expires": event_date(body, "expiration"),
            "age_days": age_days,
            "status": body.get("status", []),
            "nameservers": [ns.get("ldhName", "").lower() for ns in body.get("nameservers", [])],
        }

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return self.execute(configs, {"domain": "example.com"}, "domain_lookup")
