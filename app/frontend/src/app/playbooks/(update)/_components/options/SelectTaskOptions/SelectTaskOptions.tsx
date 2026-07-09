"use client"
import React from 'react'

import { ChevronRight } from 'lucide-react'
import { Glyph, type GlyphTone } from '@/components/soar'
import { TaskOperationType } from '../../../_providers/PlaybookOperationProvider'
import { TASK_OPERATIONS } from './resources'

// static classes so Tailwind can see them at build time
const BADGE_TONE: Record<GlyphTone, string> = {
  signal: 'bg-signal-soft text-signal-text',
  moss: 'bg-moss-soft text-moss-text',
  amber: 'bg-amber-soft text-amber-text',
  rose: 'bg-rose-soft text-rose-text',
  slate: 'bg-slate-soft text-slate-text',
}

const SelectTaskOptions: React.FC<{
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>
}> = ({ setTaskOperation }) => {
  return (
    <div className='flex h-full flex-col'>
      <div className='border-b border-line px-4 py-3.5'>
        <div className="text-[15px] font-semibold">Add a step</div>
        <div className="text-[12.5px] text-ink-faint">Choose what this step does</div>
      </div>

      <div className='flex flex-1 flex-col gap-5 overflow-auto p-4'>
        {TASK_OPERATIONS.map(group => (
          <div key={`select-task-operation-${group.label}`} className="flex flex-col gap-2">
            <div className="px-0.5 text-[11px] font-bold uppercase tracking-wider text-ink-faint">
              {group.label}
            </div>

            <div className="flex flex-col gap-2.5">
              {group.options.map(option => (
                <button
                  type="button"
                  key={`select-task-option-${option.label}`}
                  onClick={() => setTaskOperation(option.operation)}
                  className="flex items-start gap-3 rounded-md border border-line bg-card p-3 text-left transition-colors hover:border-line-strong hover:shadow-sm">
                  <Glyph icon={option.Icon} tone={option.tone} size="lg" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5 text-[13px] font-semibold text-ink">
                      {option.label}
                      {option.badge && (
                        <span className={`rounded-[5px] px-1.5 py-px text-[10px] font-bold uppercase tracking-wide ${BADGE_TONE[option.badgeTone ?? 'rose']}`}>
                          {option.badge}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-[11.5px] leading-snug text-ink-faint">
                      {option.description}
                    </p>
                  </div>
                  <ChevronRight className="mt-0.5 size-4 shrink-0 text-ink-faint" />
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default SelectTaskOptions
