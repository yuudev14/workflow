import React, { useContext, useState } from 'react'
import { Link, useParams } from 'react-router'

import { AlertTriangle, Bell, Check, Copy, KeyRound, Link2, MousePointerClick } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { toast } from '@/hooks/use-toast'
import BASE_URL from '@/settings/urls'
import {
  TRIGGER_ON_CREATE,
  TRIGGER_ON_DELETE,
  TRIGGER_WEBHOOK,
  isModuleEventTrigger,
} from '@/settings/triggers'
import { PlaybookOperationContext } from '../../../providers/PlaybookOperationProvider'

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
  referenced: {
    title: 'Referenced trigger',
    icon: <Link2 />,
    text: 'This playbook only runs when another playbook references it. There are no parameters to configure here.',
  },
}

type WebhookAuth = 'api_key' | 'none'

const CopyField: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="flex flex-col gap-1.5">
    <FieldLabel>{label}</FieldLabel>
    <div className="flex items-start gap-2 rounded-md border border-line bg-paper-sunken px-3 py-2">
      <code className="flex-1 whitespace-pre-wrap break-all font-mono text-[11.5px] text-ink-soft">{value}</code>
      <button
        type="button"
        aria-label={`Copy ${label.toLowerCase()}`}
        className="text-ink-soft hover:text-foreground"
        onClick={() => navigator.clipboard.writeText(value).then(() => toast({ title: 'Copied' }))}
      >
        <Copy className="size-3.5" />
      </button>
    </div>
  </div>
)

const WebhookTriggerParameters = () => {
  const { setPlaybookData, closeSidebar, workflowData } = useContext(PlaybookOperationContext)
  const { playbookId } = useParams() as { playbookId: string }

  // the API treats a missing auth as api_key, so the panel does too
  const savedAuth = (workflowData.trigger_parameters as { auth?: WebhookAuth } | null | undefined)?.auth
  const [auth, setAuth] = useState<WebhookAuth>(savedAuth ?? 'api_key')

  const url = new URL(`${BASE_URL.PLAYBOOK_SERVICE_API}/api/v1/webhooks/${playbookId}`, window.location.origin).href
  const curl = [
    `curl -X POST '${url}'`,
    ...(auth === 'api_key' ? [`-H 'Authorization: Bearer <API key>'`] : []),
    `-H 'Content-Type: application/json'`,
    `-d '{"host": "pc-1"}'`,
  ].join(' \\\n  ')

  const save = () => {
    setPlaybookData((data) => ({ ...data, trigger_parameters: { auth } }))
    closeSidebar()
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-line px-4 py-3.5">
        <div className="text-[15px] font-semibold">Webhook trigger</div>
        <div className="text-[12.5px] text-ink-faint">Run this playbook when an outside system calls its URL</div>
      </div>

      <div className="flex flex-1 flex-col gap-4 overflow-auto p-4">
        <div className="flex flex-col gap-1.5">
          <FieldLabel>Authentication</FieldLabel>
          <Select value={auth} onValueChange={(v) => setAuth(v as WebhookAuth)}>
            <SelectTrigger aria-label="Authentication">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-popover">
              <SelectItem value="api_key">API key</SelectItem>
              <SelectItem value="none">None</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {auth === 'api_key' ? (
          <p className="flex items-start gap-2 rounded-md border border-line bg-card px-3 py-2.5 text-[11.5px] leading-snug text-ink-soft">
            <KeyRound className="mt-0.5 size-3.5 shrink-0" />
            <span>
              Send an API key whose user can execute playbooks.{' '}
              <Link to="/settings/api-keys" className="font-semibold underline">
                Manage API keys
              </Link>
            </span>
          </p>
        ) : (
          <p className="flex items-start gap-2 rounded-md border border-amber-text/30 bg-amber-soft px-3 py-2.5 text-[11.5px] leading-snug text-amber-text">
            <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
            Anyone with this URL can run this playbook.
          </p>
        )}

        <CopyField label="Webhook URL" value={url} />
        <CopyField label="Example request" value={curl} />

        <p className="text-[11.5px] leading-snug text-ink-faint">
          The JSON body is available in steps as <code className="font-mono">{'{{ var.input.parameters }}'}</code>.
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

  if (triggerType === TRIGGER_WEBHOOK) {
    return <WebhookTriggerParameters />
  }

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
