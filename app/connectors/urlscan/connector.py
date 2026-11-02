import time

from connectors.core.connector import Connector

BASE_URL = "https://urlscan.io/api/v1"
POLL_SECONDS = 5


class URLScan(Connector):
    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        base = configs.get("base_url") or BASE_URL
        headers = {"API-Key": configs.get("api_key", "")}
        if operation == "search":
            body = self.http("GET", f"{base}/search/", headers=headers,
                             params={"q": params["query"], "size": params.get("size") or 10})
            return {"total": body.get("total", 0), "results": body.get("results", [])}
        if operation == "scan_url":
            submitted = self.http("POST", f"{base}/scan/", headers=headers, json={
                "url": params["url"], "visibility": params.get("visibility") or "unlisted",
            })
            # the result is a 404 until the scan ends
            deadline = time.monotonic() + float(params.get("max_wait_seconds") or 60)
            while True:
                time.sleep(POLL_SECONDS)
                result = self.http("GET", f"{base}/result/{submitted['uuid']}/", headers=headers, missing_ok=True)
                if result is not None:
                    overall = result.get("verdicts", {}).get("overall", {})
                    return {
                        "uuid": submitted["uuid"],
                        "result_url": submitted.get("result"),
                        "malicious": overall.get("malicious", False),
                        "score": overall.get("score", 0),
                        "categories": overall.get("categories", []),
                    }
                if time.monotonic() > deadline:
                    raise TimeoutError(f"the scan {submitted['uuid']} did not end in time")
        raise ValueError(f"operation ({operation}) does not exist in URLScan")

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return self.execute(configs, {"query": "domain:example.com", "size": 1}, "search")
