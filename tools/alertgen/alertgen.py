#!/usr/bin/env python3
"""Dev/demo alert generator for YTSoar.

Python stdlib only — no venv, no requirements.txt. Run it on the host against
the dev stack; nothing in app/ytsoar imports it.

    python3 tools/alertgen/alertgen.py seed
    python3 tools/alertgen/alertgen.py seed --count 200
    python3 tools/alertgen/alertgen.py drip
    python3 tools/alertgen/alertgen.py storm

Ingest API-key auth does not exist yet, so this authenticates as a user: it
logs in and lets an http.cookiejar carry the httpOnly cookies. Never read or
forward the token by hand — the cookies ride every later request on their own.
"""

import argparse
import datetime as dt
import http.cookiejar
import json
import os
import random
import sys
import time
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import scenarios  # noqa: E402

API = os.environ.get("YTSOAR_API", "http://localhost:8080").rstrip("/")
USERNAME = os.environ.get("ADMIN_USERNAME", "admin")
PASSWORD = os.environ.get("ADMIN_PASSWORD", "admin123!")


class Client:
    def __init__(self, base):
        self.base = base
        jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

    def request(self, method, path, body=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(
            self.base + path,
            data=data,
            method=method,
            headers={"Content-Type": "application/json"},
        )
        try:
            with self.opener.open(req, timeout=60) as resp:
                raw = resp.read()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as err:
            detail = err.read().decode(errors="replace")[:400]
            raise SystemExit(f"{method} {path} -> {err.code}\n{detail}") from None
        except urllib.error.URLError as err:
            raise SystemExit(
                f"cannot reach {self.base} ({err.reason}).\n"
                "Is the dev stack up?  docker compose -f docker/dev.docker-compose.yml up -d"
            ) from None

    def login(self):
        self.request("POST", "/api/auth/v1/login", {"username": USERNAME, "password": PASSWORD})


def spread_over_days(n, days, rng):
    """Timestamps spread across the last `days`, newest last.

    Without this every seeded alert lands in one bucket and the 14-day volume
    chart is a single spike instead of a shape. The ingest payload accepts
    created_at for exactly this reason — real forwarders send the event time too.
    """
    now = dt.datetime.now(dt.timezone.utc)
    stamps = []
    for _ in range(n):
        offset = rng.random() ** 1.6 * days  # weighted toward recent
        stamps.append(now - dt.timedelta(days=offset, seconds=rng.randint(0, 86400)))
    stamps.sort()
    return [s.strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z" for s in stamps]


def cmd_seed(client, args):
    rng = random.Random(args.seed)
    bodies = scenarios.build_covering_all_kinds(args.count, rng)

    for body, stamp in zip(bodies, spread_over_days(len(bodies), args.days, rng)):
        body["created_at"] = stamp

    if args.count >= 50:
        # One transaction, so every row shares a created_at unless the payload
        # carries its own — which is precisely the case the keyset cursor's id
        # tiebreaker exists for.
        result = client.request("POST", "/api/alerts/v1/batch", {"alerts": bodies})
        created = result.get("created", [])
        print(f"batch: {len(created)} created, {result.get('failed', 0)} failed")
        alerts = created
    else:
        alerts = []
        for i, body in enumerate(bodies, 1):
            alerts.append(client.request("POST", "/api/alerts/v1", body))
            print(f"  [{i}/{len(bodies)}] {body['severity']:<8} {body['source_kind']:<9} {body['title'][:52]}")

    seed_incidents(client, alerts, rng, args.incidents)


def seed_incidents(client, alerts, rng, count):
    """Escalate a few alerts so the incidents queue and status_mix are non-empty."""
    if not alerts or count <= 0:
        return

    picks = rng.sample(alerts, min(count, len(alerts)))
    incidents = []
    for alert in picks:
        incident = client.request("POST", f"/api/alerts/v1/{alert['id']}/escalate", {})
        incidents.append(incident)
        print(f"escalated -> incident {incident['id'][:8]}  {incident['title'][:50]}")

    if not incidents:
        return

    client.request("POST", f"/api/incidents/v1/{incidents[0]['id']}/notes", {
        "body": "Confirmed with the user over Teams. Containing the host and "
                "pulling a memory capture before we re-image.",
    })

    # Move one along the stepper so status_mix has more than one bucket.
    if len(incidents) > 1:
        client.request("PATCH", f"/api/incidents/v1/{incidents[1]['id']}/status",
                       {"status": "investigating"})
    if len(incidents) > 2:
        client.request("PATCH", f"/api/incidents/v1/{incidents[2]['id']}/status",
                       {"status": "contained"})
    print(f"seeded {len(incidents)} incidents (1 note, status spread)")


def cmd_drip(client, args):
    rng = random.Random(args.seed)
    print(f"dripping one alert every ~{args.interval}s — Ctrl-C to stop")
    n = 0
    try:
        while True:
            body = scenarios.build(rng)
            alert = client.request("POST", "/api/alerts/v1", body)
            n += 1
            print(f"  [{n}] {body['severity']:<8} {body['source_kind']:<9} "
                  f"{body['title'][:48]}  dedup={alert['dedup_count']}")
            time.sleep(rng.uniform(args.interval * 0.5, args.interval * 1.5))
    except KeyboardInterrupt:
        print(f"\nstopped after {n} alerts")


def cmd_storm(client, args):
    """N byte-identical alerts must collapse to ONE row with dedup_count=N.

    If this produces N rows instead, the unique partial index
    alerts_open_fingerprint_idx is wrong. It is also the module.events check:
    exactly one alert.created followed by N-1 alert.updated.
    """
    body = scenarios.storm_body()


    before = client.request("GET", "/api/alerts/v1?limit=1")["total"]
    print(f"sending {args.count} identical alerts ({before} alerts already in the queue)...")

    first = client.request("POST", "/api/alerts/v1", body)
    alert_id = first["id"]

    remaining = args.count - 1
    step = max(1, remaining // 10)
    for i in range(remaining):
        last = client.request("POST", "/api/alerts/v1", body)
        if (i + 1) % step == 0:
            print(f"  {i + 2}/{args.count}  dedup_count={last['dedup_count']}")

    final = client.request("GET", f"/api/alerts/v1/{alert_id}")
    after = client.request("GET", "/api/alerts/v1?limit=1")["total"]

    print(f"\nalert id        : {alert_id}")
    print(f"dedup_count     : {final['dedup_count']}  (want {args.count})")
    print(f"new rows        : {after - before}  (want 1)")
    print(f"timeline entries: {len(final['timeline'])}  (want 1 — a recurrence is not a timeline row)")

    ok = final["dedup_count"] == args.count and (after - before) == 1 and len(final["timeline"]) == 1
    print("\nPASS: the storm collapsed into one alert" if ok else
          "\nFAIL: dedup did not hold — check alerts_open_fingerprint_idx")
    return 0 if ok else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--seed", type=int, default=None, help="rng seed for reproducible data")
    sub = parser.add_subparsers(dest="mode", required=True)

    p = sub.add_parser("seed", help="a spread of alerts across all source kinds")
    p.add_argument("--count", type=int, default=40)
    p.add_argument("--days", type=int, default=14, help="spread created_at over this many days")
    p.add_argument("--incidents", type=int, default=3, help="how many alerts to escalate")

    p = sub.add_parser("drip", help="one alert at a time, for watching the UI live")
    p.add_argument("--interval", type=float, default=20.0)

    p = sub.add_parser("storm", help="N identical alerts — proves dedup")
    p.add_argument("--count", type=int, default=200)

    args = parser.parse_args()

    client = Client(API)
    client.login()
    print(f"logged in to {API} as {USERNAME}\n")

    if args.mode == "seed":
        cmd_seed(client, args)
    elif args.mode == "drip":
        cmd_drip(client, args)
    elif args.mode == "storm":
        return cmd_storm(client, args)
    return 0


if __name__ == "__main__":
    sys.exit(main() or 0)
