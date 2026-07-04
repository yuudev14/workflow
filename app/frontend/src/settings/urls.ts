const BASE_URL = {
  PLAYBOOK_SERVICE_API: (process.env.NEXT_PUBLIC_PLAYBOOK_SERVICE_API || "/playbook-api"),
  CONNECTORS_SERVICE_API: (process.env.NEXT_PUBLIC_CONNECTORS_SERVICE_API || "/connector-api"),
}

console.log(BASE_URL)

export default BASE_URL