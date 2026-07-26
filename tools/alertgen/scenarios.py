"""ECS-shaped alert scenarios for the dev/demo generator.

The ECS field names are the point, not decoration. `threat.technique.id`,
`host.hostname`, `source.ip` and `file.hash.sha256` are the exact paths that
ATT&CK extraction (#15) and IOC extraction (#12) will read, so seeding them now
makes that work testable against real data later. A future ELK branch points
these same builders at an index instead of the ingest endpoint.

Stdlib only — this file is imported by alertgen.py, which has no dependencies.
"""

import hashlib
import random

WORKSTATIONS = [
    "WIN-DC01", "WIN-DC02", "FIN-WS14", "HR-WS03", "ENG-WS21",
    "ENG-WS22", "SALES-WS07", "OPS-WS11", "LAB-WS02", "EXEC-WS01",
]

USERS = [
    "j.doe", "a.mercer", "s.okafor", "r.tanaka", "m.silva",
    "k.novak", "p.laurent", "d.ashford", "svc_backup", "administrator",
]

DOMAINS = ["contoso-mail.com", "invoices-secure.net", "sharepoint-docs.co", "dhl-tracking.info"]

EXTERNAL_IPS = [
    "185.220.101.7", "45.133.1.92", "193.176.86.24", "104.244.76.13",
    "5.188.206.18", "91.219.236.166", "37.120.222.19",
]

GEO = {
    "185.220.101.7": ("Germany", "DE"),
    "45.133.1.92": ("Netherlands", "NL"),
    "193.176.86.24": ("Russia", "RU"),
    "104.244.76.13": ("United States", "US"),
    "5.188.206.18": ("Romania", "RO"),
    "91.219.236.166": ("Ukraine", "UA"),
    "37.120.222.19": ("Switzerland", "CH"),
}


def _sha256(rng):
    return hashlib.sha256(str(rng.random()).encode()).hexdigest()


def _internal_ip(rng):
    return f"10.{rng.randint(0, 4)}.{rng.randint(0, 255)}.{rng.randint(2, 254)}"


def _host_block(rng):
    host = rng.choice(WORKSTATIONS)
    return {"hostname": host, "name": host, "ip": [_internal_ip(rng)], "os": {"family": "windows"}}


def _threat(tactic, technique_id, technique_name):
    return {
        "framework": "MITRE ATT&CK",
        "tactic": {"name": tactic},
        "technique": {"id": technique_id, "name": technique_name},
    }


# --- EDR -------------------------------------------------------------------

def encoded_powershell(rng):
    user = rng.choice(USERS)
    return {
        "title": "Encoded PowerShell command executed",
        "severity": rng.choice(["high", "high", "medium"]),
        "source_kind": "edr",
        "reporter": "CrowdStrike Falcon",
        "tags": ["execution", "powershell"],
        "payload": {
            "host": _host_block(rng),
            "user": {"name": user, "domain": "CORP"},
            "process": {
                "name": "powershell.exe",
                "pid": rng.randint(1000, 9999),
                "command_line": "powershell.exe -nop -w hidden -enc SQBFAFgAKABOAGUAdwAtAE8AYgBqAGUAYwB0AA==",
                "parent": {"name": rng.choice(["winword.exe", "excel.exe", "outlook.exe"])},
            },
            "file": {"hash": {"sha256": _sha256(rng)}},
            "threat": _threat("Execution", "T1059.001", "PowerShell"),
        },
    }


def lsass_dump(rng):
    return {
        "title": "Credential dumping via LSASS memory access",
        "severity": "critical",
        "source_kind": "edr",
        "reporter": "CrowdStrike Falcon",
        "tags": ["credential-access", "lsass"],
        "payload": {
            "host": _host_block(rng),
            "user": {"name": rng.choice(USERS), "domain": "CORP"},
            "process": {
                "name": rng.choice(["rundll32.exe", "procdump.exe", "taskmgr.exe"]),
                "pid": rng.randint(1000, 9999),
                "command_line": "rundll32.exe C:\\Windows\\System32\\comsvcs.dll, MiniDump 672 C:\\Temp\\l.dmp full",
                "target": {"name": "lsass.exe"},
            },
            "file": {"path": "C:\\Temp\\l.dmp", "hash": {"sha256": _sha256(rng)}},
            "threat": _threat("Credential Access", "T1003.001", "LSASS Memory"),
        },
    }


def mass_file_rename(rng):
    count = rng.randint(400, 9000)
    return {
        "title": f"Mass file rename detected ({count} files)",
        "severity": "critical",
        "source_kind": "edr",
        "reporter": "Microsoft Defender for Endpoint",
        "tags": ["impact", "ransomware"],
        "payload": {
            "host": _host_block(rng),
            "user": {"name": rng.choice(USERS)},
            "process": {"name": rng.choice(["explorer.exe", "wscript.exe", "unknown.exe"])},
            "file": {"extension": rng.choice([".locked", ".enc", ".crypt"]), "count": count},
            "threat": _threat("Impact", "T1486", "Data Encrypted for Impact"),
        },
    }


# --- Identity --------------------------------------------------------------

def impossible_travel(rng):
    ip_a, ip_b = rng.sample(EXTERNAL_IPS, 2)
    country_a, code_a = GEO[ip_a]
    country_b, code_b = GEO[ip_b]
    return {
        "title": f"Impossible travel: {country_a} then {country_b}",
        "severity": "high",
        "source_kind": "identity",
        "reporter": "Microsoft Entra ID Protection",
        "tags": ["initial-access", "anomaly"],
        "payload": {
            "user": {"name": rng.choice(USERS), "domain": "CORP"},
            "source": {"ip": ip_b, "geo": {"country_name": country_b, "country_iso_code": code_b}},
            "related": {"ip": [ip_a, ip_b]},
            "event": {"outcome": "success", "action": "user-login"},
            "previous_login": {"ip": ip_a, "geo": {"country_name": country_a}},
            "threat": _threat("Initial Access", "T1078", "Valid Accounts"),
        },
    }


def password_spray(rng):
    attempts = rng.randint(40, 800)
    return {
        "title": f"Password spray against {rng.randint(12, 90)} accounts",
        "severity": rng.choice(["high", "medium"]),
        "source_kind": "identity",
        "reporter": "Okta",
        "tags": ["credential-access", "bruteforce"],
        "payload": {
            "source": {"ip": rng.choice(EXTERNAL_IPS)},
            "user": {"name": rng.choice(USERS)},
            "event": {"outcome": "failure", "action": "user-authentication", "count": attempts},
            "threat": _threat("Credential Access", "T1110.003", "Password Spraying"),
        },
    }


def mfa_push_bombing(rng):
    return {
        "title": "Repeated MFA push notifications rejected",
        "severity": "high",
        "source_kind": "identity",
        "reporter": "Duo Security",
        "tags": ["credential-access", "mfa"],
        "payload": {
            "user": {"name": rng.choice(USERS)},
            "source": {"ip": rng.choice(EXTERNAL_IPS)},
            "event": {"action": "mfa-challenge", "outcome": "denied", "count": rng.randint(8, 45)},
            "threat": _threat("Credential Access", "T1621", "Multi-Factor Authentication Request Generation"),
        },
    }


# --- Email -----------------------------------------------------------------

def phishing_attachment(rng):
    domain = rng.choice(DOMAINS)
    return {
        "title": "Phishing email with macro-enabled attachment",
        "severity": rng.choice(["high", "medium"]),
        "source_kind": "email",
        "reporter": "Proofpoint TAP",
        "tags": ["initial-access", "phishing"],
        "payload": {
            "user": {"name": rng.choice(USERS)},
            "email": {
                "from": {"address": f"billing@{domain}"},
                "to": {"address": f"{rng.choice(USERS)}@corp.local"},
                "subject": rng.choice([
                    "Outstanding invoice #48213",
                    "Payroll adjustment - action required",
                    "Your package could not be delivered",
                ]),
            },
            "file": {"name": "invoice_48213.xlsm", "hash": {"sha256": _sha256(rng)}},
            "threat": _threat("Initial Access", "T1566.001", "Spearphishing Attachment"),
        },
    }


def credential_harvest_link(rng):
    domain = rng.choice(DOMAINS)
    return {
        "title": "Credential harvesting link clicked",
        "severity": "high",
        "source_kind": "email",
        "reporter": "Microsoft Defender for Office 365",
        "tags": ["initial-access", "phishing"],
        "payload": {
            "user": {"name": rng.choice(USERS)},
            "email": {"from": {"address": f"no-reply@{domain}"}},
            "url": {"full": f"https://{domain}/o365/login?id={_sha256(rng)[:12]}", "domain": domain},
            "event": {"action": "url-click", "outcome": "success"},
            "threat": _threat("Initial Access", "T1566.002", "Spearphishing Link"),
        },
    }


# --- Firewall --------------------------------------------------------------

def c2_beaconing(rng):
    ip = rng.choice(EXTERNAL_IPS)
    country, code = GEO[ip]
    return {
        "title": "Periodic beaconing to rare external destination",
        "severity": "critical",
        "source_kind": "firewall",
        "reporter": "Palo Alto NGFW",
        "tags": ["command-and-control", "beaconing"],
        "payload": {
            "host": _host_block(rng),
            "source": {"ip": _internal_ip(rng)},
            "destination": {
                "ip": ip,
                "port": rng.choice([443, 8443, 4444, 8080]),
                "geo": {"country_name": country, "country_iso_code": code},
            },
            "network": {"protocol": "tls", "bytes": rng.randint(2000, 80000)},
            "event": {"interval_seconds": rng.choice([30, 60, 300])},
            "threat": _threat("Command and Control", "T1071.001", "Web Protocols"),
        },
    }


def outbound_port_scan(rng):
    return {
        "title": f"Outbound port scan across {rng.randint(200, 4000)} ports",
        "severity": rng.choice(["medium", "low"]),
        "source_kind": "firewall",
        "reporter": "Palo Alto NGFW",
        "tags": ["discovery", "scanning"],
        "payload": {
            "host": _host_block(rng),
            "source": {"ip": _internal_ip(rng)},
            "destination": {"ip": rng.choice(EXTERNAL_IPS), "port": rng.randint(1, 65535)},
            "network": {"protocol": "tcp", "bytes": rng.randint(500, 9000)},
            "threat": _threat("Discovery", "T1046", "Network Service Discovery"),
        },
    }


# --- DLP -------------------------------------------------------------------

def bulk_cloud_upload(rng):
    mb = rng.randint(80, 6000)
    return {
        "title": f"Bulk upload to personal cloud storage ({mb} MB)",
        "severity": rng.choice(["medium", "low"]),
        "source_kind": "dlp",
        "reporter": "Forcepoint DLP",
        "tags": ["exfiltration", "dlp"],
        "payload": {
            "host": _host_block(rng),
            "user": {"name": rng.choice(USERS)},
            "destination": {"domain": rng.choice(["dropbox.com", "drive.google.com", "wetransfer.com"])},
            "network": {"bytes": mb * 1024 * 1024},
            "file": {"count": rng.randint(20, 900), "extension": rng.choice([".zip", ".pdf", ".xlsx"])},
            "threat": _threat("Exfiltration", "T1048", "Exfiltration Over Alternative Protocol"),
        },
    }


SCENARIOS = [
    encoded_powershell,
    lsass_dump,
    mass_file_rename,
    impossible_travel,
    password_spray,
    mfa_push_bombing,
    phishing_attachment,
    credential_harvest_link,
    c2_beaconing,
    outbound_port_scan,
    bulk_cloud_upload,
]

SOURCE_KINDS = ["edr", "identity", "email", "firewall", "dlp"]


def build(rng=None):
    """One random alert body, ready to POST."""
    rng = rng or random
    return rng.choice(SCENARIOS)(rng)


def build_covering_all_kinds(count, rng=None):
    """`count` alerts guaranteed to touch every source_kind and every severity.

    A purely random draw can leave a source_kind or a severity empty, which
    makes the summary charts look broken when they are merely unlucky.
    """
    rng = rng or random
    out = []

    for fn in SCENARIOS:
        out.append(fn(rng))

    severities = ["critical", "high", "medium", "low"]
    for i, severity in enumerate(severities):
        alert = SCENARIOS[i % len(SCENARIOS)](rng)
        alert["severity"] = severity
        out.append(alert)

    while len(out) < count:
        out.append(build(rng))

    return out[:count]


def storm_body():
    """One byte-identical alert, for proving dedup.

    Deterministic on purpose: same source_kind, reporter, title and primary
    entity means the same fingerprint every time, so the ingest path must
    collapse the whole storm into a single row.
    """
    return {
        "title": "Periodic beaconing to rare external destination",
        "severity": "critical",
        "source_kind": "firewall",
        "reporter": "Palo Alto NGFW",
        "tags": ["command-and-control", "beaconing"],
        "payload": {
            "host": {"hostname": "OPS-WS11", "name": "OPS-WS11"},
            "source": {"ip": "10.0.4.77"},
            "destination": {"ip": "185.220.101.7", "port": 8443},
            "network": {"protocol": "tls", "bytes": 4096},
            "threat": _threat("Command and Control", "T1071.001", "Web Protocols"),
        },
    }
