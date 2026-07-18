import React, { useContext, useState } from 'react'

import { AlertTriangle, Bell, Check, Link2, MousePointerClick, Webhook } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import {
  TRIGGER_ON_CREATE,
  TRIGGER_ON_DELETE,
  isModuleEventTrigger,
} from '@/settings/triggers'
import { PlaybookOperationContext } from '../../../_providers/PlaybookOperationProvider'

type Module = 'alert' | 'incident'

const FieldLabel: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <label className="text-[11px] font-semibold uppercase tracking-wide text-ink-soft">
    {children}
  </label>
)

// summary copy for the triggers with nothing to configure
const SIMPLE_TRIGGERS: Record<string, { title: string; icon: React.ReactNode; text: string }> = {
  manual: {
    title: 'Manual trigger',
    icon: <MousePointerClick />,
    text: 'This playbook runs on demand from the Playbooks list or the API. There are no trigger parameters to configure.',
  },
  webhook: {
    title: 'Webhook trigger',
    icon: <Webhook />,
    text: "This playbook fires when an external system calls its unique webhook URL. There are no parameters to configure here.",
  },
  referenced: {
    title: 'Referenced trigger',
    icon: <Link2 />,
    text: 'This playbook only runs when another playbook references it. There are no parameters to configure here.',
  },
}

const PlaybookTriggerParameters = () => {
  const { currentNode, setNodes, setPlaybookData, closeSidebar, workflowData } = useContext(
    PlaybookOperationContext
  )

  const triggerType = workflowData.trigger_type
  const isModuleEvent = isModuleEventTrigger(triggerType)

  // the event itself is carried by the trigger type; only the module is a parameter
  const savedModule = (workflowData.trigger_parameters as { module?: Module } | null | undefined)
    ?.module
  const [module, setModule] = useState<Module>(savedModule ?? 'alert')

  const eventLabel =
    triggerType === TRIGGER_ON_CREATE
      ? 'created'
      : triggerType === TRIGGER_ON_DELETE
        ? 'deleted'
        : 'updated'

  if (!isModuleEvent) {
    const summary = SIMPLE_TRIGGERS[triggerType ?? ''] ?? SIMPLE_TRIGGERS.manual
    return (
      <div className="flex h-full flex-col">
        <div className="border-b border-line px-4 py-3.5">
          <div className="text-[15px] font-semibold">{summary.title}</div>
          <div className="text-[12.5px] text-ink-faint">How this playbook starts</div>
        </div>
        <div className="flex flex-1 flex-col gap-3 p-4">
          <div className="flex items-start gap-3 rounded-md border border-line bg-card p-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-[10px] bg-slate-soft text-slate-text [&_svg]:size-4">
              {summary.icon}
            </span>
            <p className="text-[12.5px] leading-snug text-ink-soft">{summary.text}</p>
          </div>
        </div>
      </div>
    )
  }

  const save = () => {
    // the module rides along with the playbook in trigger_parameters
    setPlaybookData((data) => ({ ...data, trigger_parameters: { module } }))
    setNodes((nodes) =>
      nodes.map((node) =>
        node.id === currentNode?.id
          ? {
              ...node,
              data: {
                ...node.data,
                label: `${module} · ${eventLabel}`,
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
          Run this playbook when a record is {eventLabel}
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

        <p className="rounded-md border border-dashed border-line-strong bg-paper-sunken px-3 py-2.5 text-[11.5px] leading-snug text-ink-faint">
          This playbook will run every time an{' '}
          <span className="font-semibold capitalize text-ink-soft">{module}</span> is{' '}
          <span className="font-semibold text-ink-soft">{eventLabel}</span>.
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
