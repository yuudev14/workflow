import { ConnectorInfo } from "@/services/connectors/connectors.schema"
import { useContext, useState } from "react"
import { WorkflowOperationContext } from "../../../_providers/WorkflowOperationProvider"
import OperationLayout from "../../OperationLayout"
import ConnectorOperation from "./ConnectorOperation"
import ConnectorList from "./ConnectorList"

const ConnectorsOption = () => {
  
  const { setTaskOperation, connector, setConnector } = useContext(WorkflowOperationContext)
  if (connector === null) return (
    <OperationLayout backHandler={() => {
      setConnector(null)
      setTaskOperation(null)
    }}>
      <ConnectorList setConnector={setConnector} />
    </OperationLayout>
  )
  return (
    <OperationLayout backHandler={() => {
      setConnector(null)

    }}>
      <ConnectorOperation connector={connector} />
    </OperationLayout>
  )
}

export default ConnectorsOption