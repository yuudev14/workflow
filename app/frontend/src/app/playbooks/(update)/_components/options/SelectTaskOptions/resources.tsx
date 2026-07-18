import { CircleCheckBig, Code, GitBranch, LayoutGrid, PencilLineIcon, Timer, Wrench } from 'lucide-react'
import { TaskOperationGroup } from './schema'

export const TASK_OPERATIONS: readonly TaskOperationGroup[] = [
  {
    label: "Execute",
    options: [
      {
        label: "Connectors",
        description: "Slack, Jira, VirusTotal and every other installed third-party connector",
        tone: "signal",
        Icon: LayoutGrid,
        operation: "connector"
      },
      {
        label: "Code snippet",
        description: "A quick custom transform between steps — no connector to install",
        tone: "moss",
        badge: "python",
        badgeTone: "signal",
        Icon: Code,
        operation: "code"
      },
      {
        label: "Code snippet",
        description: "Same, in JavaScript. Plain JS only — TypeScript syntax won't run here",
        tone: "amber",
        badge: "node",
        badgeTone: "moss",
        Icon: Code,
        operation: "code_js"
      },
      {
        label: "Utilities",
        description: "HTTP Request and other in-worker builtins — no sandbox subprocess",
        tone: "slate",
        badge: "in-worker",
        badgeTone: "slate",
        Icon: Wrench,
        operation: "utility"
      },
    ]
  },
  {
    label: "Evaluate",
    options: [
      {
        label: "Condition",
        description: "If / else if / else — branch the playbook on a compare or expression",
        tone: "rose",
        Icon: GitBranch,
        operation: "decision"
      },
      {
        label: "Wait",
        description: "Pause for a fixed duration, or until a condition becomes true",
        tone: "amber",
        badge: "new",
        Icon: Timer,
        operation: "wait"
      },
      {
        label: "Approval",
        description: "Hold the run until an analyst approves or rejects it",
        tone: "moss",
        Icon: CircleCheckBig,
        operation: "approval"
      },
      {
        label: "Input Prompt",
        description: "Collect a value from an operator before continuing",
        tone: "signal",
        Icon: PencilLineIcon,
        operation: "input_prompt"
      },
    ]
  },
] as const