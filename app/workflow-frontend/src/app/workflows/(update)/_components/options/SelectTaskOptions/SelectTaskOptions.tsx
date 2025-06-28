"use client"
import React from 'react'

import OptionButton from '@/components/buttons/OptionButton'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { TaskOperationType } from '../../../_providers/WorkflowOperationProvider'
import { TASK_OPERATIONS } from './resources'
import { Info } from 'lucide-react'




const SelectTaskOptions: React.FC<{
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>
}> = ({ setTaskOperation }) => {
  return (
    <div className='flex flex-col gap-3 h-full pt-5'>
      <div className='px-3 flex justify-between items-center'>
        <Label className="uppercase text-lg">
          Select Task Operation
        </Label>
      </div>

      <div className='flex-1 overflow-auto px-3 pb-5'>
        {TASK_OPERATIONS.map(operation => (
          <div key={`select-task-operation-${operation.label}`}>
            <Label className="text-base">
              {operation.label}
            </Label>
            <Separator />
            <div className="flex flex-col gap-3">
              {operation.options.map(option => (
                <OptionButton
                  key={`select-task-option-${option.label}`}
                  Icon={option.Icon}
                  iconClass={operation.iconClass}
                  buttonClass='h-[64px]'
                  onClick={() => setTaskOperation(option.operation)}
                >
                  <div className='flex-1 flex gap-2 items-center h-full'>
                    <Label className='capitalize text'>{option.label}</Label>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild><Info size={14} /></TooltipTrigger>
                        <TooltipContent>
                          {option.tooltip}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>

                </OptionButton>
              ))}
            </div>

          </div>
        ))}
      </div>
    </div>
  )
}

export default SelectTaskOptions