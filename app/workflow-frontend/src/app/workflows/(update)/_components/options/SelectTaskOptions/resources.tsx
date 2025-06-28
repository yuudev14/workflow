import { Cable, CircleCheckBig, Code, Network, PencilLineIcon, Timer, Wrench } from 'lucide-react'
import { TaskOperationGroup } from './schema'

export const TASK_OPERATIONS: readonly TaskOperationGroup[] = [
  {
    label: "Execute",
    options: [
      {
        label: "Connector",
        tooltip: <p>Add to library</p>,
        Icon: Cable,
        operation: "connector"
      },
      {
        label: "Utility",
        tooltip: <p>Add to library</p>,
        Icon: Wrench,
        operation: "utility"
      },
      {
        label: "Code",
        tooltip: <p>Add to library</p>,
        Icon: Code,
        operation: "code"
      },
    ]
  },
  {
    label: "Evaluate",
    iconClass: 'bg-quaternary text-quaternary-foreground',
    options: [
      {
        label: "Decision",
        tooltip: <p>Add to library</p>,
        Icon: Network,
        operation: "decision"
      },
      {
        label: "Wait",
        tooltip: <p>Add to library</p>,
        Icon: Timer,
        operation: "wait"
      },
      {
        label: "Approval",
        tooltip: <p>Add to library</p>,
        Icon: CircleCheckBig,
        operation: "approval"
      },
      {
        label: "Input Prompt",
        tooltip: <p>Add to library</p>,
        Icon: PencilLineIcon,
        operation: "input_prompt"
      },
    ]
  },
] as const