import { useContext } from "react"
import { PlaybookOperationContext } from "../../../_providers/PlaybookOperationProvider"
import OperationLayout from "../../OperationLayout"
import ConnectorOperation from "./ConnectorOperation"
import ConnectorList from "./ConnectorList"

const ConnectorsOption = () => {
  
  const { setTaskOperation, connector, setConnector } = useContext(PlaybookOperationContext)
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