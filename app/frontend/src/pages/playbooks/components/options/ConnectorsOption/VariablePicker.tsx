import { useContext, useRef } from "react";
import { Braces } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { FLOW_START_ID } from "@/settings/reactFlowIds";
import { PlaybookOperationContext } from "../../../providers/PlaybookOperationProvider";

type Variable = { label: string; ref: string; field?: boolean };

// Steps that run before the current one, nearest first, with their declared output fields.
const useEarlierSteps = (): Variable[][] => {
  const { currentNode, nodes, edges, connectorQuery } = useContext(PlaybookOperationContext);
  if (!currentNode) return [];

  const seen = new Set<string>([currentNode.id]);
  const queue = [currentNode.id];
  const ancestors: string[] = [];
  while (queue.length) {
    const id = queue.shift()!;
    for (const edge of edges) {
      if (edge.target !== id || seen.has(edge.source)) continue;
      seen.add(edge.source);
      queue.push(edge.source);
      ancestors.push(edge.source);
    }
  }

  return ancestors.flatMap((id) => {
    const data = nodes.find((node) => node.id === id)?.data;
    if (!data?.name || data.name === FLOW_START_ID || !data.connector_id) return [];
    // bracket keys with JSON quoting work in jinja2, nunjucks, Python dicts and JS objects
    const step = `steps[${JSON.stringify(data.name)}]`;
    const output = connectorQuery?.data
      ?.find((connector) => connector.id === data.connector_id)
      ?.operations.find((op) => op.annotation === data.operation)?.output;
    return [[
      { label: data.name, ref: step },
      ...(output ?? []).map((field) => ({
        label: field.name,
        ref: `${step}[${JSON.stringify(field.name)}]`,
        field: true,
      })),
    ]];
  });
};

export const VariablePicker: React.FC<{ onPick: (ref: string) => void }> = ({ onPick }) => {
  const steps = useEarlierSteps();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        type="button"
        aria-label="Insert variable"
        className="rounded-sm p-1 text-ink-faint hover:bg-signal-soft hover:text-signal-text">
        <Braces className="size-3.5" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="max-h-80 overflow-auto bg-popover"
        // the caller puts the focus back in its own field
        onCloseAutoFocus={(e) => e.preventDefault()}>
        <DropdownMenuLabel className="text-[11px] uppercase tracking-wide text-ink-faint">
          Earlier steps
        </DropdownMenuLabel>
        {steps.length === 0 && (
          <div className="px-2 py-1.5 text-[12px] text-ink-faint">No earlier steps</div>
        )}
        {steps.flat().map((v) => (
          <DropdownMenuItem
            key={v.ref}
            onSelect={() => onPick(v.ref)}
            className={v.field ? "pl-6 font-mono text-[11.5px]" : "font-semibold"}>
            {v.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

// A text input whose picker inserts `{{ var.<ref> }}` at the cursor.
export const VariableInput: React.FC<
  Omit<React.ComponentProps<"input">, "onChange" | "value"> & {
    value: string;
    onChange: (value: string) => void;
    inputClassName?: string;
  }
> = ({ value, onChange, className, inputClassName, ...props }) => {
  const input = useRef<HTMLInputElement>(null);

  const insert = (ref: string) => {
    const text = `{{ var.${ref} }}`;
    const start = input.current?.selectionStart ?? value.length;
    const end = input.current?.selectionEnd ?? value.length;
    onChange(value.slice(0, start) + text + value.slice(end));
    requestAnimationFrame(() => {
      input.current?.focus();
      input.current?.setSelectionRange(start + text.length, start + text.length);
    });
  };

  return (
    <div className={cn("relative", className)}>
      <Input
        ref={input}
        className={cn(inputClassName, "pr-8")}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        {...props}
      />
      <div className="absolute inset-y-0 right-1 flex items-center">
        <VariablePicker onPick={insert} />
      </div>
    </div>
  );
};
