import { Incident, IncidentsSummary } from "./incidents.schema";

const INC_241_TIMELINE = [
  {
    title: "Alert triggered — Outbound connection to known C2 IP",
    detail: "14:26:41 · IDS/firewall",
    tone: "rose" as const,
  },
  {
    title: "Playbook ran — IOC Enrichment & Block",
    detail: "14:26:44 · failed, rule template missing",
    tone: "signal" as const,
  },
  {
    title: "Escalated to incident by n.reyes",
    detail: "14:31:02 · 2 related alerts attached",
    tone: "amber" as const,
  },
  {
    title: "Host isolated manually",
    detail: "14:44:18 · n.reyes · via EDR console",
    tone: "moss" as const,
  },
];

export const INCIDENTS: Incident[] = [
  {
    id: "INC-241",
    title: "INC-241 — C2 beaconing on finance subnet",
    severity: "critical",
    status: "investigating",
    owner: "n.reyes",
    age: "41m",
    openedAgo: "41m ago",
    slaLeft: "19m left",
    alertCount: 3,
    runCount: 2,
    linkedAlerts: [
      { id: "a-3", title: "Outbound connection to known C2 IP", source: "IDS/firewall", severity: "critical" },
      { id: "a-1", title: "Suspicious PowerShell execution on WIN-DESKTOP-04", source: "EDR", severity: "critical" },
      { id: "a-6", title: "DNS tunneling pattern detected", source: "Firewall / IDS", severity: "high" },
    ],
    runs: [
      { playbook: "IOC Enrichment & Block — run #4469", detail: "Failed at Firewall Block · 0:08s · 41m ago", outcome: "failed" },
      { playbook: "Malware Sandbox Detonation — run #4468", detail: "Host quarantined · 2m 11s · 38m ago", outcome: "success" },
    ],
    timeline: [
      ...INC_241_TIMELINE,
      { title: "Awaiting firewall rule fix to close out", detail: "in progress" },
    ],
    notes: [
      {
        who: "n.reyes",
        when: "14:31",
        text: "Escalating — same C2 IP hit two hosts within 5 minutes of each other. Isolating WIN-DESKTOP-04 manually while the firewall rule template gets fixed.",
      },
      {
        who: "takakiiiyuuu",
        when: "14:47",
        text: "Rule template issue is in the connector config, not the firewall itself — pushing a fix now.",
      },
    ],
    iocs: [
      { type: "IP", value: "203.0.113.44" },
      { type: "Domain", value: "x9f-relay.top" },
      { type: "Host", value: "WIN-DESKTOP-04" },
      { type: "Hash", value: "a1c4…9e02" },
    ],
    tags: ["finance-subnet", "c2-beacon"],
  },
  {
    id: "INC-240",
    title: "INC-240 — Credential stuffing campaign",
    severity: "high",
    status: "contained",
    owner: "takakiiiyuuu",
    age: "2h",
    openedAgo: "2h ago",
    slaLeft: "1h 40m left",
    alertCount: 8,
    runCount: 1,
    tags: ["identity"],
  },
  {
    id: "INC-238",
    title: "INC-238 — Phishing campaign (Q3 payroll lure)",
    severity: "medium",
    status: "resolved",
    owner: "n.reyes",
    age: "1d",
    openedAgo: "1d ago",
    alertCount: 14,
    runCount: 1,
    tags: ["phishing"],
  },
  {
    id: "INC-235",
    title: "INC-235 — Malware on WIN-DESKTOP-04",
    severity: "high",
    status: "resolved",
    owner: "takakiiiyuuu",
    age: "2d",
    openedAgo: "2d ago",
    alertCount: 2,
    runCount: 1,
    tags: ["endpoint"],
  },
];

export const INCIDENTS_SUMMARY: IncidentsSummary = {
  openTotal: 6,
  statusMix: [
    { status: "investigating", count: 2 },
    { status: "contained", count: 1 },
    { status: "resolved", count: 3 },
  ],
  severityMix: [
    { severity: "critical", count: 1 },
    { severity: "high", count: 3 },
    { severity: "medium", count: 2 },
  ],
  mttrTrend: [30, 40, 34, 52, 46, 64, 58, 78].map((v) => 100 - v),
  slaAtRisk: [
    { id: "INC-241", title: "INC-241 — C2 beaconing", left: "19m left" },
    { id: "INC-240", title: "INC-240 — Credential stuffing", left: "1h 40m left" },
    { id: "INC-233", title: "INC-233 — Unpatched CVE alert", left: "breached 4h ago", breached: true },
  ],
};
