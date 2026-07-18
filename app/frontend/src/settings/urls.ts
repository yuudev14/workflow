const BASE_URL = {
  PLAYBOOK_SERVICE_API: (process.env.NEXT_PUBLIC_PLAYBOOK_SERVICE_API || "/playbook-api"),
  CONNECTORS_SERVICE_API: (process.env.NEXT_PUBLIC_CONNECTORS_SERVICE_API || "/connector-api"),
}

console.log(BASE_URL)

// Live status socket. Derived from the page origin so it works behind nginx
// (ws://host/ws/playbook) without extra config; override with an env var if the
// api is reached directly. Client-only — only call from effects/hooks.
export const getPlaybookWsUrl = (): string => {
  if (process.env.NEXT_PUBLIC_PLAYBOOK_WS_URL) {
    return process.env.NEXT_PUBLIC_PLAYBOOK_WS_URL;
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws/playbook`;
};

export default BASE_URL