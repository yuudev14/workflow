import React, { useContext } from 'react'

import { useQuery } from '@tanstack/react-query'
import { ChevronRight, MousePointerClick, ToggleRight, Webhook, type LucideIcon } from 'lucide-react'
import PlaybookService from '@/services/playbooks/playbooks'
import { PlaybookTriggerType } from '@/services/playbooks/playbooks.schema'
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from '@/settings/reactFlowIds'
import { Glyph, type GlyphTone } from '@/components/soar'
import { PlaybookOperationContext } from '../../../_providers/PlaybookOperationProvider'

// per-trigger presentation. Unknown backend triggers fall back to a neutral look.
const TRIGGER_META: Record<string, { icon: LucideIcon; tone: GlyphTone; description: string }> = {
  manual: {
    icon: MousePointerClick,
    tone: 'slate',
    description: 'Run on demand from the Playbooks list or the API — no automatic trigger',
  },
  webhook: {
    icon: Webhook,
    tone: 'signal',
    description: "Fire when an external system calls this playbook's unique URL",
  },
}

// Module-event trigger — fires off an Alert/Incident event. Phase 1 client-only
// option (backend wiring is Phase 2); configured in PlaybookTriggerParameters.
const MODULE_EVENT_ID = 'module_event'

const SelectPlaybookTriggerOption = () => {
  const { setPlaybookData, setOpenOperationSidebar, setNodes, setCurrentNode } =
    useContext(PlaybookOperationContext)

  const triggerTypesQuery = useQuery({
    queryKey: ['workflow-trigger-type-lists'],
    queryFn: async () => PlaybookService.getPlaybookTriggerTypes(),
  })

  // create the start node for the chosen trigger. For module events we keep the
  // sidebar open on the start node so its module/event can be configured.
  const applyTrigger = (triggerId: string, label: string, openConfig: boolean) => {
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
        <div className="flex items-center gap-1.5 text-[13px] font-semibold capitalize text-ink">
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
        {triggerTypesQuery.data?.map((trigger: PlaybookTriggerType) => {
          const meta = TRIGGER_META[trigger.name] ?? {
            icon: MousePointerClick,
            tone: 'slate' as GlyphTone,
            description: trigger.description ?? 'Start this playbook.',
          }
          return (
            <Tile
              key={`trigger-type-${trigger.id}`}
              icon={meta.icon}
              tone={meta.tone}
              name={trigger.name}
              description={meta.description}
              onClick={() => applyTrigger(trigger.id, trigger.name, false)}
            />
          )
        })}

        <Tile
          icon={ToggleRight}
          tone="rose"
          name="Module event"
          badge="new"
          description="Fire automatically when an Alert or Incident is created or updated"
          onClick={() => applyTrigger(MODULE_EVENT_ID, 'Module event', true)}
        />
      </div>
    </div>
  )
}

export default SelectPlaybookTriggerOption
