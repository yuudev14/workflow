import React, { useCallback, useContext } from 'react'

import OptionButton from '@/components/buttons/OptionButton'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { MousePointerClick, Webhook } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import WorkflowService from '@/services/worfklows/workflows'
import { WorkflowTriggerType } from '@/services/worfklows/workflows.schema'
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from '@/settings/reactFlowIds'
import { WorkflowOperationContext } from '../../../_providers/WorkflowOperationProvider'

const SelectWorkflowTriggerOption = () => {

  const { setWorkflowData, setOpenOperationSidebar, setNodes } = useContext(WorkflowOperationContext)
  const triggerTypesQuery = useQuery({
    queryKey: ['workflow-trigger-type-lists'],
    queryFn: async () => {
      return await WorkflowService.getWorkflowTriggerTypes()
    }
  })

  const renderIcon = useCallback((triggerName: string) => {
    switch (triggerName) {
      case "manual":
        return MousePointerClick
      case "webhook":
        return Webhook
      default:
        return MousePointerClick // for now
    }
  }, [])

  const selectTriggerType = (trigger: WorkflowTriggerType) => {
    setWorkflowData(workflow => ({ ...workflow, trigger_type: trigger.id }))
    setNodes(nodes => {
      return nodes.filter(_node => _node.id !== FLOW_SELECT_TRIGGER_ID).concat({
        id: FLOW_START_ID,
        data: {
          name: FLOW_START_ID,
          label: "manual"
        },
        position: { x: 100, y: 100 },
        type: "start",
        draggable: true,
        
      })
    })
    setOpenOperationSidebar(false)
  }

  return (
    <div className='flex flex-col gap-3 px-3 py-5'>
      <Label className="uppercase text-lg">
        Choose a trigger
      </Label>
      <Separator />

      <div className="flex flex-col gap-3">
        {triggerTypesQuery.data?.map(trigger => (
          <OptionButton Icon={renderIcon(trigger.name)} key={`trigger-type-${trigger.id}`} onClick={() => selectTriggerType(trigger)}>
            <div>
              <Label className='uppercase text-base'>{trigger.name}</Label>
              <p>Small description about manual</p>
            </div>

          </OptionButton>
        ))}
      </div>
    </div>
  )
}

export default SelectWorkflowTriggerOption