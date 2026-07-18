import React, { useContext } from 'react'

import {
  ChevronRight,
  Link2,
  MousePointerClick,
  PenLine,
  Plus,
  Trash2,
  Webhook,
  type LucideIcon,
} from 'lucide-react'
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from '@/settings/reactFlowIds'
import {
  TRIGGER_MANUAL,
  TRIGGER_ON_CREATE,
  TRIGGER_ON_DELETE,
  TRIGGER_ON_UPDATE,
  TRIGGER_REFERENCED,
  TRIGGER_WEBHOOK,
  isModuleEventTrigger,
} from '@/settings/triggers'
import { Glyph, type GlyphTone } from '@/components/soar'
import { PlaybookOperationContext } from '../../../_providers/PlaybookOperationProvider'

// static trigger catalog for the picker — ids must match the backend enum
// (see settings/triggers.ts). Module-event triggers open the config sidebar
// so the module can be chosen right away.
const TRIGGERS: {
  id: string
  name: string
  icon: LucideIcon
  tone: GlyphTone
  description: string
  badge?: string
}[] = [
  {
    id: TRIGGER_MANUAL,
    name: 'Manual',
    icon: MousePointerClick,
    tone: 'slate',
    description: 'Run on demand from the Playbooks list or the API — no automatic trigger',
  },
  {
    id: TRIGGER_WEBHOOK,
    name: 'Webhook',
    icon: Webhook,
    tone: 'signal',
    description: "Fire when an external system calls this playbook's unique URL",
  },
  {
    id: TRIGGER_REFERENCED,
    name: 'Referenced',
    icon: Link2,
    tone: 'slate',
    description: 'Run only when another playbook references this one as a sub-playbook',
  },
  {
    id: TRIGGER_ON_CREATE,
    name: 'On create',
    icon: Plus,
    tone: 'moss',
    badge: 'new',
    description: 'Fire automatically when a record in the chosen module is created',
  },
  {
    id: TRIGGER_ON_UPDATE,
    name: 'On update',
    icon: PenLine,
    tone: 'amber',
    badge: 'new',
    description: 'Fire automatically when a record in the chosen module is updated',
  },
  {
    id: TRIGGER_ON_DELETE,
    name: 'On delete',
    icon: Trash2,
    tone: 'rose',
    badge: 'new',
    description: 'Fire automatically when a record in the chosen module is deleted',
  },
]

const SelectPlaybookTriggerOption = () => {
  const { setPlaybookData, setOpenOperationSidebar, setNodes, setCurrentNode } =
    useContext(PlaybookOperationContext)

  // create the start node for the chosen trigger. Module-event triggers keep the
  // sidebar open on the start node so their module can be configured.
  const applyTrigger = (triggerId: string, label: string) => {
    const openConfig = isModuleEventTrigger(triggerId)
    setPlaybookData((workflow) => ({ ...workflow, trigger_type: triggerId }))
    const startNode = {
      id: FLOW_START_ID,
      data: { name: FLOW_START_ID, label },
      position: { x: 100, y: 100 },
      type: 'startNode' as const,
      draggable: true,
    }
    setNodes((nodes) =>
      nodes.filter((_node) => _node.id !== FLOW_SELECT_TRIGGER_ID).concat(startNode)
    )
    if (openConfig) {
      setCurrentNode(startNode)
      setOpenOperationSidebar(true)
    } else {
      setOpenOperationSidebar(false)
    }
  }

  const Tile: React.FC<{
    icon: LucideIcon
    tone: GlyphTone
    name: string
    description: string
    badge?: string
    onClick: () => void
  }> = ({ icon, tone, name, description, badge, onClick }) => (
    <button
      type="button"
      onClick={onClick}
      className="flex items-start gap-3 rounded-md border border-line bg-card p-3 text-left transition-colors hover:border-line-strong hover:shadow-sm">
      <Glyph icon={icon} tone={tone} size="lg" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 text-[13px] font-semibold text-ink">
          {name}
          {badge && (
            <span className="rounded-[5px] bg-rose-soft px-1.5 py-px text-[10px] font-bold uppercase tracking-wide text-rose-text">
              {badge}
            </span>
          )}
        </div>
        <p className="mt-1 text-[11.5px] leading-snug text-ink-faint">{description}</p>
      </div>
      <ChevronRight className="mt-0.5 size-4 shrink-0 text-ink-faint" />
    </button>
  )

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-line px-4 py-3.5">
        <div className="text-[15px] font-semibold">Select trigger</div>
        <div className="text-[12.5px] text-ink-faint">How should this playbook start?</div>
      </div>

      <div className="flex flex-1 flex-col gap-2.5 overflow-auto p-4">
        {TRIGGERS.map((trigger) => (
          <Tile
            key={`trigger-type-${trigger.id}`}
            icon={trigger.icon}
            tone={trigger.tone}
            name={trigger.name}
            description={trigger.description}
            badge={trigger.badge}
            onClick={() => applyTrigger(trigger.id, trigger.name)}
          />
        ))}
      </div>
    </div>
  )
}

export default SelectPlaybookTriggerOption
