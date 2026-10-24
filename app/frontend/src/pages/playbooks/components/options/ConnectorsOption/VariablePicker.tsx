import { useContext, useRef, useState } from "react";
import { ChevronRight, GripVertical, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { FLOW_START_ID } from "@/settings/reactFlowIds";
import { PlaybookOperationContext } from "../../../providers/PlaybookOperationProvider";

type Node = { label: string; ref: string; type?: string; children: Node[] };

// a dot on a dict key that is also a method name gives the method in jinja2
const DICT_METHODS = new Set(["keys", "values", "items", "get", "pop", "update", "copy", "clear", "setdefault"]);

const child = (ref: string, key: string) =>
  /^[A-Za-z_]\w*$/.test(key) && !DICT_METHODS.has(key) ? `${ref}.${key}` : `${ref}[${JSON.stringify(key)}]`;

const fields = (ref: string, list: { name: string; type?: string }[]): Node[] =>
  list.map((f) => ({ label: f.name, ref: child(ref, f.name), type: f.type, children: [] }));

const RECORD_FIELDS: Record<"alert" | "incident", [string, string][]> = {
  alert: [
    ["id", "string"], ["title", "string"], ["severity", "string"], ["status", "string"], ["source_kind", "string"],
    ["reporter", "string"], ["tags", "array"], ["payload", "object"], ["fingerprint", "string"], ["created_at", "string"],
  ],
  incident: [["id", "string"], ["title", "string"], ["severity", "string"], ["status", "string"], ["tags", "array"], ["created_at", "string"]],
};

// Steps that run before the current one, nearest first, then the run input.
const useVariables = (): { steps: Node[]; input: Node[] } => {
  const { currentNode, nodes, edges, connectorQuery, workflowData } = useContext(PlaybookOperationContext);
  if (!currentNode) return { steps: [], input: [] };

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

  const steps = ancestors.flatMap((id) => {
    const data = nodes.find((n) => n.id === id)?.data;
    const name = data?.name;
    if (!name || name === FLOW_START_ID || !data.connector_id) return [];
    const output = connectorQuery?.data
      ?.find((c) => c.id === data.connector_id)
      ?.operations.find((op) => op.annotation === data.operation)?.output;
    const ref = child("var.steps", name);
    return [{ label: name, ref, type: data.connector_name ?? undefined, children: fields(ref, output ?? []) }];
  });

  const module = (workflowData.trigger_parameters as { module?: "alert" | "incident" } | null | undefined)?.module ?? "alert";
  const record = "var.input.records[0]";
  return {
    steps,
    input: [
      { label: "parameters", ref: "var.input.parameters", type: "object", children: [] },
      { label: "module_type", ref: "var.input.module_type", type: "string", children: [] },
      {
        label: "records[0]",
        ref: record,
        type: module,
        children: fields(record, RECORD_FIELDS[module].map(([name, type]) => ({ name, type }))),
      },
    ],
  };
};

// Keeps the nodes whose path has the query, with their parents.
const filter = (nodes: Node[], q: string): Node[] =>
  nodes.flatMap((n) => {
    const children = filter(n.children, q);
    return n.ref.toLowerCase().includes(q) || children.length ? [{ ...n, children }] : [];
  });

// The field that a click in the panel writes to: the text input or code editor that had the focus last.
// One step form is open at a time, so one shared slot is enough.
export const variableTarget: { current: ((text: string) => void) | null } = { current: null };

const Row: React.FC<{ n: Node; depth: number; open: (ref: string) => boolean; toggle: (ref: string) => void }> = ({
  n,
  depth,
  open,
  toggle,
}) => {
  const text = `{{ ${n.ref} }}`;
  const isOpen = open(n.ref);
  return (
    <>
      <div
        className="group flex items-center gap-1 rounded-sm pr-2 text-[12px] hover:bg-paper-sunken"
        style={{ paddingLeft: depth * 12 + 4 }}>
        <button
          type="button"
          aria-label={isOpen ? `Close ${n.label}` : `Open ${n.label}`}
          className={cn("flex size-4 shrink-0 items-center justify-center text-ink-faint", !n.children.length && "invisible")}
          onClick={() => toggle(n.ref)}>
          <ChevronRight className={cn("size-3.5 transition-transform", isOpen && "rotate-90")} />
        </button>
        <button
          type="button"
          draggable
          title={text}
          aria-label={`Insert ${n.ref}`}
          onDragStart={(e) => e.dataTransfer.setData("text/plain", text)}
          // keep the focus and the cursor in the field
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => variableTarget.current?.(text)}
          className={cn("flex min-w-0 flex-1 cursor-grab items-center gap-1.5 py-1 text-left", depth ? "font-mono text-[11.5px]" : "font-semibold")}>
          <GripVertical className="size-3 shrink-0 text-ink-faint opacity-0 group-hover:opacity-100" />
          <span className="truncate">{n.label}</span>
          {n.type && (
            <span className="ml-auto shrink-0 rounded border border-line px-1 font-mono text-[10px] font-normal text-ink-faint">{n.type}</span>
          )}
        </button>
      </div>
      {isOpen && n.children.map((c) => <Row key={c.ref} n={c} depth={depth + 1} open={open} toggle={toggle} />)}
    </>
  );
};

// Docked beside the step form: search, click to insert at the cursor, or drag into any field.
export const VariablesPanel: React.FC = () => {
  const { steps, input } = useVariables();
  const [query, setQuery] = useState("");
  const [toggled, setToggled] = useState<Set<string>>(new Set());
  const q = query.trim().toLowerCase();
  const shown = { steps: filter(steps, q), input: filter(input, q) };
  // the nearest step starts open, and a search opens every match
  const open = (ref: string) => !!q || toggled.has(ref) !== (ref === steps[0]?.ref);
  const toggle = (ref: string) =>
    setToggled((refs) => {
      const next = new Set(refs);
      if (!next.delete(ref)) next.add(ref);
      return next;
    });

  const label = "px-2 pb-1 pt-3 text-[11px] uppercase tracking-wide text-ink-faint";
  const empty = "px-2 py-1 text-[12px] text-ink-faint";
  return (
    <aside
      aria-label="Variables"
      className="absolute left-full top-0 flex h-full w-[300px] flex-col border-r border-line bg-card shadow-[12px_0_30px_rgba(20,22,26,0.06)]">
      <div className="border-b border-line p-3">
        <div className="mb-2 text-[13px] font-semibold">Variables</div>
        <div className="relative">
          <Search className="absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-ink-faint" />
          <Input
            placeholder="Search variables"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="h-8 pl-7 text-[12.5px]"
          />
        </div>
      </div>
      <div className="flex-1 overflow-auto px-1 pb-3">
        <div className={label}>Earlier steps</div>
        {shown.steps.length === 0 && <div className={empty}>{q ? "No match" : "No earlier steps"}</div>}
        {shown.steps.map((n) => <Row key={n.ref} n={n} depth={0} open={open} toggle={toggle} />)}
        <div className={label}>Run input</div>
        {shown.input.length === 0 && <div className={empty}>No match</div>}
        {shown.input.map((n) => <Row key={n.ref} n={n} depth={1} open={open} toggle={toggle} />)}
      </div>
      <p className="border-t border-line bg-paper-sunken px-3 py-2 text-[11px] text-ink-faint">
        Click a variable to insert it in the last field you used, or drag it into a field.
      </p>
    </aside>
  );
};

// A text input that takes the panel's inserts at its cursor.
export const VariableInput: React.FC<
  Omit<React.ComponentProps<"input">, "onChange" | "value"> & {
    value: string;
    onChange: (value: string) => void;
    inputClassName?: string;
  }
> = ({ value, onChange, className, inputClassName, ...props }) => {
  const input = useRef<HTMLInputElement>(null);
  // an insert comes after later edits, so it must call the newest onChange
  const latest = useRef(onChange);
  latest.current = onChange;

  const insert = (text: string) => {
    const el = input.current;
    if (!el) return;
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    latest.current(el.value.slice(0, start) + text + el.value.slice(end));
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(start + text.length, start + text.length);
    });
  };

  return (
    <Input
      ref={input}
      className={cn(className, inputClassName)}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onFocus={() => (variableTarget.current = insert)}
      {...props}
    />
  );
};
