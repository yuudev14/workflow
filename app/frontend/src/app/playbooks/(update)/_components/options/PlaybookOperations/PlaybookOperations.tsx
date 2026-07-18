import React, { useContext } from "react"
import SelectTaskOptions from "../SelectTaskOptions"
import { X } from "lucide-react"
import PlaybookTriggerParameters from "../PlaybookTriggerParameters/PlaybookTriggerParameters"
import {
  CODE_SNIPPET_JS_ID,
  CODE_SNIPPET_PY_ID,
  CONDITION_CONNECTOR_ID,
  FLOW_SELECT_TRIGGER_ID,
  FLOW_START_ID,
} from "@/settings/reactFlowIds"
import { ConnectorsOption } from "../ConnectorsOption"
import ConnectorOperation from "../ConnectorsOption/ConnectorOperation"
import OperationLayout from "../../OperationLayout"
import SelectPlaybookTriggerOption from "../SelectPlaybookTriggerOption"
import { TaskOperationType, PlaybookOperationContext } from "../../../_providers/PlaybookOperationProvider"


const PlaybookOperations: React.FC = () => {
  const {
    hasTriggerStep,
    isNewNode,
    currentNode,
    setNodes,
    taskOperation,
    closeSidebar,
    setTaskOperation,
    setEdges
  } = useContext(PlaybookOperationContext)

  const cancelHandler = () => {

    if (currentNode) {
      setNodes((nodes) => {
        if (currentNode == null || [FLOW_START_ID, FLOW_SELECT_TRIGGER_ID].includes(currentNode.id) || !isNewNode) {
          return nodes
        }
        return nodes.filter(_node => _node.id !== currentNode.id)
      })
    }
    closeSidebar()
    if(currentNode){
      setEdges(edgs => edgs.filter(edg => edg.id !== currentNode.id))
    }
  }


  return (

    <div className='absolute left-0 top-0 z-50 flex h-[calc(100vh-64px)] w-full max-w-[460px] flex-col border-r border-line bg-card shadow-[12px_0_30px_rgba(20,22,26,0.08)]'>
      <button
        className="absolute right-4 top-4 z-10 flex size-7 items-center justify-center rounded-sm text-ink-soft hover:bg-paper-sunken hover:text-foreground"
        onClick={cancelHandler}
      >
        <X size={16} />
      </button>
      {!hasTriggerStep && currentNode?.id == FLOW_SELECT_TRIGGER_ID ? (
        <SelectPlaybookTriggerOption />
      ) : hasTriggerStep && currentNode?.data.name == FLOW_START_ID ? (
        <PlaybookTriggerParameters />
      ) : taskOperation === null ? (
        <SelectTaskOptions setTaskOperation={setTaskOperation} />
      ) : (

        <OperationWindow taskOperation={taskOperation} setTaskOperation={setTaskOperation} />

      )

      }
      {/* <SelectPlaybookTrigger /> */}
      {/* <SelectTaskOptions /> */}

      {/* <ConnectorOperation /> */}
      {/* <ConnectorList /> */}

    </div>

  )
}

// step types that map to a single, known connector id — these skip the connector
// picker and drop straight into the config form for that virtual connector.
const DIRECT_CONNECTOR: Partial<Record<NonNullable<TaskOperationType>, string>> = {
  decision: CONDITION_CONNECTOR_ID,
  code: CODE_SNIPPET_PY_ID,
  code_js: CODE_SNIPPET_JS_ID,
}

// Renders the config form for a fixed connector id (condition / code snippets),
// resolved from the already-loaded connector list.
const DirectConnectorOperation: React.FC<{
  connectorId: string
  onBack: () => void
}> = ({ connectorId, onBack }) => {
  const { connectorQuery } = useContext(PlaybookOperationContext)
  const connector = connectorQuery?.data?.find((c) => c.id === connectorId)

  if (connectorQuery?.isLoading)
    return <div className="p-4 text-[13px] text-ink-faint">Loading…</div>
  if (!connector)
    return (
      <div className="p-4 text-[13px] text-ink-faint">
        The <span className="font-mono">{connectorId}</span> connector isn&apos;t installed.
      </div>
    )

  return (
    <OperationLayout backHandler={onBack}>
      <ConnectorOperation connector={connector} />
    </OperationLayout>
  )
}

const OperationWindow: React.FC<{
  taskOperation: TaskOperationType,
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>
}> = ({ taskOperation, setTaskOperation }) => {

  if (taskOperation === "connector") return <ConnectorsOption />

  const directId = taskOperation ? DIRECT_CONNECTOR[taskOperation] : undefined
  if (directId)
    return (
      <DirectConnectorOperation
        connectorId={directId}
        onBack={() => setTaskOperation(null)}
      />
    )

  return null
}


export default PlaybookOperations