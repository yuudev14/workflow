"use client"
import React from 'react'

import OptionButton from '@/components/buttons/OptionButton'
import { Label } from '@/components/ui/label'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { TaskOperationType } from '../../../_providers/PlaybookOperationProvider'
import { TASK_OPERATIONS } from './resources'
import { Info } from 'lucide-react'




const SelectTaskOptions: React.FC<{
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>
}> = ({ setTaskOperation }) => {
  return (
    <div className='flex h-full flex-col'>
      <div className='border-b border-line px-4 py-3.5'>
        <div className="text-[15px] font-semibold">Add a step</div>
        <div className="text-[12.5px] text-ink-faint">Choose what this step does</div>
      </div>

      <div className='flex flex-1 flex-col gap-4 overflow-auto p-4'>
        {TASK_OPERATIONS.map(operation => (
          <div key={`select-task-operation-${operation.label}`} className="flex flex-col gap-2">
            <Label className="text-[12px] font-semibold uppercase tracking-wide text-ink-faint">
              {operation.label}
            </Label>

            <div className="flex flex-col gap-2">
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