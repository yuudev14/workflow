import React, { useContext, useState } from 'react'

import { AlertTriangle, Bell, Check, MousePointerClick, Webhook } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { PlaybookOperationContext } from '../../../_providers/PlaybookOperationProvider'

type Module = 'alert' | 'incident'
type Event = 'created' | 'updated'

const FieldLabel: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <label className="text-[11px] font-semibold uppercase tracking-wide text-ink-soft">
    {children}
  </label>
)

const PlaybookTriggerParameters = () => {
  const { currentNode, setNodes, closeSidebar, workflowData } = useContext(
    PlaybookOperationContext
  )

  const isModuleEvent = workflowData.trigger_type === 'module_event'

  const [module, setModule] = useState<Module>(
    (currentNode?.data?.trigger_module as Module) ?? 'alert'
  )
  const [event, setEvent] = useState<Event>(
    (currentNode?.data?.trigger_event as Event) ?? 'created'
  )

  // Non-module triggers (manual / webhook) have nothing to configure — show a
  // short summary instead of an empty panel.
  if (!isModuleEvent) {
    const manual = workflowData.trigger_type !== 'webhook'
    return (
      <div className="flex h-full flex-col">
        <div className="border-b border-line px-4 py-3.5">
          <div className="text-[15px] font-semibold capitalize">
            {manual ? 'Manual' : 'Webhook'} trigger
          </div>
          <div className="text-[12.5px] text-ink-faint">How this playbook starts</div>
        </div>
        <div className="flex flex-1 flex-col gap-3 p-4">
          <div className="flex items-start gap-3 rounded-md border border-line bg-card p-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-[10px] bg-slate-soft text-slate-text [&_svg]:size-4">
              {manual ? <MousePointerClick /> : <Webhook />}
            </span>
            <p className="text-[12.5px] leading-snug text-ink-soft">
              {manual
                ? 'This playbook runs on demand from the Playbooks list or the API. There are no trigger parameters to configure.'
                : "This playbook fires when an external system calls its unique webhook URL. There are no parameters to configure here."}
            </p>
          </div>
        </div>
      </div>
    )
  }

  const save = () => {
    setNodes((nodes) =>
      nodes.map((node) =>
        node.id === currentNode?.id
          ? {
              ...node,
              data: {
                ...node.data,
                trigger_module: module,
                trigger_event: event,
                label: `${module} · ${event}`,
              },
            }
          : node
      )
    )
    closeSidebar()
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-line px-4 py-3.5">
        <div className="flex items-center gap-2">
          <span className="text-[15px] font-semibold">Module event</span>
          <span className="rounded-[5px] bg-rose-soft px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-rose-text">
            new
          </span>
        </div>
        <div className="text-[12.5px] text-ink-faint">
          Run this playbook off an Alert or Incident event
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-4 overflow-auto p-4">
        <div className="flex flex-col gap-1.5">
          <FieldLabel>Module</FieldLabel>
          <Select value={module} onValueChange={(v) => setModule(v as Module)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-popover">
              <SelectItem value="alert">
                <span className="flex items-center gap-2">
                  <Bell className="size-3.5 text-rose-text" />
                  Alert
                </span>
              </SelectItem>
              <SelectItem value="incident">
                <span className="flex items-center gap-2">
                  <AlertTriangle className="size-3.5 text-amber-text" />
                  Incident
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <FieldLabel>Event</FieldLabel>
          <div className="flex w-full gap-1 rounded-sm border border-line bg-paper-sunken p-1">
            {(['created', 'updated'] as Event[]).map((e) => (
              <button
                key={e}
                type="button"
                onClick={() => setEvent(e)}
                className={cn(
                  'flex-1 rounded-[6px] px-3 py-1.5 text-center text-[12px] font-semibold capitalize transition-colors',
                  event === e ? 'bg-card text-ink shadow-sm' : 'text-ink-soft hover:text-ink'
                )}>
                On {e === 'created' ? 'create' : 'update'}
              </button>
            ))}
          </div>
        </div>

        <p className="rounded-md border border-dashed border-line-strong bg-paper-sunken px-3 py-2.5 text-[11.5px] leading-snug text-ink-faint">
          This playbook will run every time an{' '}
          <span className="font-semibold capitalize text-ink-soft">{module}</span> is{' '}
          <span className="font-semibold text-ink-soft">{event}</span>.
        </p>
      </div>

      <footer className="border-t border-line bg-card p-3">
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={closeSidebar}>
            Close
          </Button>
          <Button type="button" onClick={save}>
            <Check /> Save trigger
          </Button>
        </div>
      </footer>
    </div>
  )
}

export default PlaybookTriggerParameters
