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

    <div className='absolute flex flex-col max-w-[500px] w-full bg-background border-r border-r-border h-[calc(100vh-64px)] top-0 left-0 z-50'>
      <button className="absolute top-5 right-5" onClick={cancelHandler}>
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