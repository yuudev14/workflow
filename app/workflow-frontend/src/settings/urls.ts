const BASE_URL = {
  WORKFLOW_SERVICE_API: (process.env.NEXT_PUBLIC_WORKFLOW_SERVICE_API || "/workflow-api"),
  CONNECTORS_SERVICE_API: (process.env.NEXT_PUBLIC_CONNECTORS_SERVICE_API || "/connector-api"),
}

console.log(BASE_URL)

export default BASE_URL