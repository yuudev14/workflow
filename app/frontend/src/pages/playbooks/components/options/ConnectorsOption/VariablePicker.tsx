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

// a dot on a dict key that is also a method name gives the method in jinja2
const DICT_METHODS = new Set(["keys", "values", "items", "get", "pop", "update", "copy", "clear", "setdefault"]);

// `var.steps.lookup.ip`, with JSON-quoted brackets only for keys that a dot cannot reach
const path = (...keys: string[]) =>
  keys.reduce(
    (ref, key) =>
      /^[A-Za-z_]\w*$/.test(key) && !DICT_METHODS.has(key) ? `${ref}.${key}` : `${ref}[${JSON.stringify(key)}]`,
    "var",
  );

const RECORD_FIELDS = {
  alert: ["id", "title", "severity", "status", "source_kind", "reporter", "tags", "payload", "fingerprint", "created_at"],
  incident: ["id", "title", "severity", "status", "tags", "created_at"],
};

// What `var.input` holds: the records the run was started on and the analyst's parameters.
const useRunInput = (): Variable[] => {
  const { workflowData } = useContext(PlaybookOperationContext);
  const module = (workflowData.trigger_parameters as { module?: "alert" | "incident" } | null | undefined)?.module;
  const record = `${path("input", "records")}[0]`;
  return [
    { label: "parameters", ref: path("input", "parameters") },
    { label: "module_type", ref: path("input", "module_type") },
    { label: "records[0]", ref: record },
    ...RECORD_FIELDS[module ?? "alert"].map((field) => ({ label: field, ref: `${record}.${field}`, field: true })),
  ];
};

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
    const name = data?.name;
    if (!name || name === FLOW_START_ID || !data.connector_id) return [];
    const output = connectorQuery?.data
      ?.find((connector) => connector.id === data.connector_id)
      ?.operations.find((op) => op.annotation === data.operation)?.output;
    return [[
      { label: name, ref: path("steps", name) },
      ...(output ?? []).map((field) => ({
        label: field.name,
        ref: path("steps", name, field.name),
        field: true,
      })),
    ]];
  });
};

// onPick gets the whole template, for example `{{ var.steps.lookup.ip }}`.
export const VariablePicker: React.FC<{ onPick: (text: string) => void }> = ({ onPick }) => {
  const steps = useEarlierSteps();
  const input = useRunInput();
  const item = (v: Variable) => (
    <DropdownMenuItem
      key={v.ref}
      onSelect={() => onPick(`{{ ${v.ref} }}`)}
      className={v.field ? "pl-6 font-mono text-[11.5px]" : "font-semibold"}>
      {v.label}
    </DropdownMenuItem>
  );
  const label = "text-[11px] uppercase tracking-wide text-ink-faint";
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
        <DropdownMenuLabel className={label}>Earlier steps</DropdownMenuLabel>
        {steps.length === 0 && (
          <div className="px-2 py-1.5 text-[12px] text-ink-faint">No earlier steps</div>
        )}
        {steps.flat().map(item)}
        <DropdownMenuLabel className={label}>Run input</DropdownMenuLabel>
        {input.map(item)}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

// A text input whose picker inserts the template at the cursor.
export const VariableInput: React.FC<
  Omit<React.ComponentProps<"input">, "onChange" | "value"> & {
    value: string;
    onChange: (value: string) => void;
    inputClassName?: string;
  }
> = ({ value, onChange, className, inputClassName, ...props }) => {
  const input = useRef<HTMLInputElement>(null);

  const insert = (text: string) => {
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
