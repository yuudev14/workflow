import React, { useContext } from "react"
import SelectTaskOptions from "../SelectTaskOptions"
import { X } from "lucide-react"
import PlaybookTriggerParameters from "../PlaybookTriggerParameters/PlaybookTriggerParameters"
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from "@/settings/reactFlowIds"
import { ConnectorsOption } from "../ConnectorsOption"
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

const OperationWindow: React.FC<{
  taskOperation: TaskOperationType,
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>
}> = ({ taskOperation }) => {

  switch (taskOperation) {
    case "connector":
      return <ConnectorsOption />
  }
  return null
}


export default PlaybookOperations