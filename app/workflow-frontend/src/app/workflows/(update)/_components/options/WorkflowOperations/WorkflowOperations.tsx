import React, { useContext } from "react"
import SelectTaskOptions from "../SelectTaskOptions"
import { X } from "lucide-react"
import WorkFlowTriggerParameters from "../WorkflowTriggerParameters/WorkFlowTriggerParameters"
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from "@/settings/reactFlowIds"
import { ConnectorsOption } from "../ConnectorsOption"
import SelectWorkflowTriggerOption from "../SelectWorkflowTriggerOption"
import { TaskOperationType, WorkflowOperationContext } from "../../../_providers/WorkflowOperationProvider"


const WorkflowOperations: React.FC = () => {
  const {
    hasTriggerStep,
    isNewNode,
    currentNode,
    setNodes,
    taskOperation,
    closeSidebar,
    setTaskOperation
  } = useContext(WorkflowOperationContext)

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
  }


  return (

    <div className='absolute flex flex-col max-w-[500px] w-full bg-background border-r border-r-border h-[calc(100vh-64px)] top-0 left-0 z-50'>
      <button className="absolute top-5 right-5" onClick={cancelHandler}>
        <X size={16} />
      </button>
      {!hasTriggerStep && currentNode?.id == FLOW_SELECT_TRIGGER_ID ? (
        <SelectWorkflowTriggerOption />
      ) : hasTriggerStep && currentNode?.id == FLOW_START_ID ? (
        <WorkFlowTriggerParameters />
      ) : taskOperation === null ? (
        <SelectTaskOptions setTaskOperation={setTaskOperation} />
      ) : (

        <OperationWindow taskOperation={taskOperation} setTaskOperation={setTaskOperation} />

      )

      }
      {/* <SelectWorkflowTrigger /> */}
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


export default WorkflowOperations