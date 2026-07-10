import { useContext, useEffect, useRef, useState } from "react";
import {
  ConnectorInfo,
  Operation,
} from "@/services/connectors/connectors.schema";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { PlaybookOperationContext } from "../../../_providers/PlaybookOperationProvider";
import { Textarea } from "@/components/ui/textarea";
import {
  CODE_SNIPPET_JS_ID,
  CODE_SNIPPET_PY_ID,
  CONDITION_CONNECTOR_ID,
  CONDITION_OUTPUT_HANDLE,
} from "@/settings/reactFlowIds";

import CodeMirror from '@uiw/react-codemirror';
import { python } from '@codemirror/lang-python';
import { javascript } from '@codemirror/lang-javascript';
import { useTheme } from "next-themes";
import { connectorGlyph, Glyph } from "@/components/soar";
import { GripVertical, Plus, X } from "lucide-react";
import { cn } from "@/lib/utils";

const taskFormSchema = z.object({
  name: z.string().min(2, {
    message: "name must be at least 2 characters.",
  }),
  description: z.string().optional(),
  parameters: z.record(z.any()).nullish(),
  config: z.string().nullable().optional().nullish(),
  connector_name: z.string(),
  connector_id: z.string(),
  operation: z.string(),
});

// value used by the "none" option to unassign a branch
const UNASSIGN = "__none__";

const newCaseId = () =>
  typeof crypto !== "undefined" && crypto.randomUUID
    ? crypto.randomUUID()
    : `case-${Date.now()}-${Math.random().toString(36).slice(2)}`;

// route chips cycle through the status palette so each branch reads distinctly,
// matching the artifact's colored per-case destinations.
const ROUTE_TONES = ["moss", "signal", "rose", "amber", "slate"] as const;
const routeStyle = (tone: (typeof ROUTE_TONES)[number], routed: boolean) =>
  routed
    ? { color: `var(--${tone}-text)`, borderColor: `var(--${tone}-dot)` }
    : undefined;

// Inline chip that routes one branch to a destination node. Options are the
// nodes already wired to this condition node; picking one stamps that branch's
// handle onto the edge so the executor follows it for that branch. Rendered as
// the `.route` chip on the right of a case row.
const BranchRouter: React.FC<{
  handleId: string;
  label?: string;
  tone?: (typeof ROUTE_TONES)[number];
}> = ({ handleId, label = "then", tone = "signal" }) => {
  const { currentNode, nodes, edges, setEdges } = useContext(
    PlaybookOperationContext
  );
  if (!currentNode) return null;

  const outgoing = edges.filter((edge) => edge.source === currentNode.id);
  const nodeName = (id: string) =>
    nodes.find((node) => node.id === id)?.data?.name ?? id;
  const selected = outgoing.find((edge) => edge.sourceHandle === handleId);

  const assign = (target: string) => {
    setEdges((eds) =>
      eds.map((edge) => {
        if (edge.source !== currentNode.id) return edge;
        if (target !== UNASSIGN && edge.target === target)
          return { ...edge, sourceHandle: handleId, label };
        // release whichever edge previously held this branch
        if (edge.sourceHandle === handleId)
          return { ...edge, sourceHandle: CONDITION_OUTPUT_HANDLE, label: undefined };
        return edge;
      })
    );
  };

  if (outgoing.length === 0)
    return (
      <span className="ml-auto rounded-sm border border-dashed border-line-strong px-2.5 py-1.5 text-[11px] font-medium text-ink-faint">
        connect a destination first
      </span>
    );

  return (
    <Select value={selected?.target ?? ""} onValueChange={assign}>
      <SelectTrigger
        className="ml-auto h-auto w-auto min-w-0 gap-1.5 rounded-sm border-line-strong bg-paper-sunken px-2.5 py-1.5 text-[11.5px] font-semibold [&>span]:truncate"
        style={routeStyle(tone, !!selected)}>
        <SelectValue placeholder="Route to…" />
      </SelectTrigger>
      <SelectContent className="bg-popover">
        <SelectItem value={UNASSIGN}>— none —</SelectItem>
        {outgoing.map((edge) => (
          <SelectItem value={edge.target} key={`route-${edge.id}`}>
            {nodeName(edge.target)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

const CaseCard: React.FC<{
  keyword: string;
  onRemove: () => void;
  children: React.ReactNode;
  route: React.ReactNode;
}> = ({ keyword, onRemove, children, route }) => (
  <div className="flex flex-col gap-2 rounded-md border border-line bg-paper p-2.5">
    <div className="flex items-center gap-2">
      <GripVertical className="size-3.5 shrink-0 cursor-grab text-ink-faint" />
      <span className="min-w-[46px] text-[11px] font-bold text-ink">{keyword}</span>
      <button
        type="button"
        onClick={onRemove}
        className="ml-auto rounded-md p-1 text-ink-faint hover:bg-rose-soft hover:text-rose-text">
        <X className="size-3.5" />
      </button>
    </div>
    <div className="flex flex-wrap items-center gap-1.5 pl-6">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">{children}</div>
      {route}
    </div>
  </div>
);

const AddCaseButton: React.FC<{ label: string; onClick: () => void }> = ({
  label,
  onClick,
}) => (
  <button
    type="button"
    onClick={onClick}
    className="flex items-center justify-center gap-1.5 rounded-md border-[1.5px] border-dashed border-line-strong py-2.5 text-[12px] font-semibold text-ink-soft transition-colors hover:border-signal-dot hover:bg-signal-soft hover:text-signal-text">
    <Plus className="size-3.5" />
    {label}
  </button>
);

const ElseCard: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div className="flex flex-col gap-2 rounded-md border border-dashed border-line-strong bg-paper p-2.5">
    <div className="flex items-center gap-2">
      <span className="w-3.5 shrink-0" />
      <span className="min-w-[46px] text-[11px] font-bold text-ink">Else</span>
      <span className="text-[11px] text-ink-faint">no case above matched</span>
    </div>
    <div className="flex flex-wrap items-center gap-1.5 pl-6">
      <div className="flex-1" />
      {children}
    </div>
  </div>
);

// One switch branch: a truthy expression plus a stable id naming its edge handle.
type Case = { id: string; expression: string };

// Editor for the "switch_expression" operation: an ordered list of if / else-if
// expressions, each routed to its own destination, plus a trailing else. Cases
// are evaluated top-to-bottom and the first truthy one wins.
const CasesEditor: React.FC<{
  value?: Case[];
  placeholder?: string;
  onChange: (cases: Case[]) => void;
}> = ({ value, placeholder, onChange }) => {
  const { currentNode, setEdges } = useContext(PlaybookOperationContext);
  const cases: Case[] = Array.isArray(value) ? value : [];

  const update = (index: number, expression: string) =>
    onChange(cases.map((c, i) => (i === index ? { ...c, expression } : c)));
  const add = () => onChange([...cases, { id: newCaseId(), expression: "" }]);
  const remove = (index: number) => {
    const removed = cases[index];
    onChange(cases.filter((_, i) => i !== index));
    // drop the removed branch's edge back to unassigned
    if (removed && currentNode)
      setEdges((eds) =>
        eds.map((edge) =>
          edge.source === currentNode.id && edge.sourceHandle === removed.id
            ? { ...edge, sourceHandle: CONDITION_OUTPUT_HANDLE, label: undefined }
            : edge
        )
      );
  };

  return (
    <div className="flex flex-col gap-2">
      {cases.map((c, i) => (
        <CaseCard
          key={c.id}
          keyword={i === 0 ? "If" : "Else if"}
          onRemove={() => remove(i)}
          route={
            <BranchRouter
              handleId={c.id}
              tone={ROUTE_TONES[i % ROUTE_TONES.length]}
            />
          }>
          <Input
            className="min-w-[120px] flex-1 font-mono text-[11.5px]"
            placeholder={placeholder ?? '{{ var.steps["…"].field }}'}
            value={c.expression ?? ""}
            onChange={(e) => update(i, e.target.value)}
          />
        </CaseCard>
      ))}
      <AddCaseButton label='Add "else if" case' onClick={add} />
      <ElseCard>
        <BranchRouter handleId="else" tone="slate" />
      </ElseCard>
    </div>
  );
};

// One simple condition: left/operator/right plus a stable id naming its handle.
type Condition = { id: string; left: string; operator: string; right: string };

const OPERATORS = ["==", "!=", ">", "<", ">=", "<=", "contains", "not_contains"];

// Editor for the simple "switch": ordered left/operator/right conditions, each
// routed to its own destination, plus a trailing else. First match wins.
const ConditionsEditor: React.FC<{
  value?: Condition[];
  placeholder?: string;
  onChange: (conditions: Condition[]) => void;
}> = ({ value, placeholder, onChange }) => {
  const { currentNode, setEdges } = useContext(PlaybookOperationContext);
  const conditions: Condition[] = Array.isArray(value) ? value : [];

  const update = (index: number, patch: Partial<Condition>) =>
    onChange(conditions.map((c, i) => (i === index ? { ...c, ...patch } : c)));
  const add = () =>
    onChange([
      ...conditions,
      { id: newCaseId(), left: "", operator: "==", right: "" },
    ]);
  const remove = (index: number) => {
    const removed = conditions[index];
    onChange(conditions.filter((_, i) => i !== index));
    if (removed && currentNode)
      setEdges((eds) =>
        eds.map((edge) =>
          edge.source === currentNode.id && edge.sourceHandle === removed.id
            ? { ...edge, sourceHandle: CONDITION_OUTPUT_HANDLE, label: undefined }
            : edge
        )
      );
  };

  const fieldCls =
    "rounded-sm border border-line-strong bg-paper-sunken px-2 py-1.5 font-mono text-[11.5px] text-ink outline-none focus:border-signal-dot";

  return (
    <div className="flex flex-col gap-2">
      {conditions.map((c, i) => (
        <CaseCard
          key={c.id}
          keyword={i === 0 ? "If" : "Else if"}
          onRemove={() => remove(i)}
          route={
            <BranchRouter
              handleId={c.id}
              tone={ROUTE_TONES[i % ROUTE_TONES.length]}
            />
          }>
          <input
            className={cn(fieldCls, "min-w-[120px] flex-1")}
            placeholder={placeholder ?? "left"}
            value={c.left ?? ""}
            onChange={(e) => update(i, { left: e.target.value })}
          />
          <select
            className={cn(fieldCls, "shrink-0 font-semibold text-ink-soft")}
            value={c.operator}
            onChange={(e) => update(i, { operator: e.target.value })}>
            {OPERATORS.map((op) => (
              <option key={op} value={op}>
                {op}
              </option>
            ))}
          </select>
          <input
            className={cn(fieldCls, "w-[72px]")}
            placeholder="value"
            value={c.right ?? ""}
            onChange={(e) => update(i, { right: e.target.value })}
          />
        </CaseCard>
      ))}
      <AddCaseButton label="Add case" onClick={add} />
      <ElseCard>
        <BranchRouter handleId="else" tone="slate" />
      </ElseCard>
    </div>
  );
};

const ConnectorOperation: React.FC<{ connector: ConnectorInfo }> = ({
  connector,
}) => {
  const [currentOperation, setCurrentOperation] = useState<Operation | null>(
    null
  );
  const { currentNode, setNodes, closeSidebar } = useContext(
    PlaybookOperationContext
  );
  const { resolvedTheme } = useTheme();
  const glyph = connectorGlyph(connector.name);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [cachedParameter, setCachedParameter] = useState<Record<string, any>>(
    {}
  );

  // "code_snippet" / "code_snippet_js" / "condition" are single-purpose virtual
  // connectors — no saved configs, and the operation is implied rather than
  // picked from a list, so their forms skip the config/operation selectors.
  const isCondition = connector.id === CONDITION_CONNECTOR_ID;
  const isJsCode = connector.id === CODE_SNIPPET_JS_ID;
  const isCode = connector.id === CODE_SNIPPET_PY_ID || isJsCode;
  const isVirtual = isCondition || isCode;

  const taskForm = useForm<z.infer<typeof taskFormSchema>>({
    resolver: zodResolver(taskFormSchema),
    defaultValues: (() => {
      const isSameConnector =
        connector.name === currentNode?.data?.connector_name;
      const task = currentNode?.data;

      return {
        name: task?.name ?? "",
        description: isSameConnector ? task?.description ?? "" : "",
        connector_name: connector.name,
        connector_id: connector.id,
        operation: isSameConnector ? task?.operation ?? "" : undefined,
        parameters: isSameConnector ? task?.parameters : undefined,
        config: isSameConnector ? task?.config : "",
      };
    })(),
  });

  const operationName = taskForm.watch("operation");
  const parameters = taskForm.watch("parameters");

  // Virtual connectors don't ask the user to pick an operation — default to the
  // first one (code snippets have a single op; condition defaults to "switch").
  useEffect(() => {
    if (isVirtual && !operationName && connector.operations[0]) {
      taskForm.setValue("operation", connector.operations[0].annotation);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // use to check if it's first render
  const isFirstRender = useRef<boolean>(true);
  useEffect(() => {
    if (operationName) {
      const matchedOperation = connector.operations.find(
        (ops) => ops.annotation === operationName
      );
      setCurrentOperation(matchedOperation ?? null);
      taskForm.setValue(
        "parameters",
        isFirstRender.current
          ? currentNode?.data?.parameters
          : cachedParameter[operationName]
      );
    }
    isFirstRender.current = false;
  }, [operationName]);

  useEffect(() => {
    if (operationName) {
      setCachedParameter((value) => ({
        ...value,
        [operationName]: parameters,
      }));
    }
  }, [parameters, operationName]);

  const onSubmit = (val: z.infer<typeof taskFormSchema>) => {
    setNodes((nodes) =>
      nodes.map((node) => {
        if (!currentNode) return node;
        if (node.id === currentNode.id) {
          return {
            ...node,
            // the condition builtin renders as the branching node
            type:
              val.connector_id === CONDITION_CONNECTOR_ID
                ? "conditionNode"
                : node.type,
            data: { ...node.data, ...val },
          };
        }
        return node;
      })
    );
    closeSidebar();
    setCachedParameter({});
  };

  return (
    <Form {...taskForm}>
      <form
        onSubmit={taskForm.handleSubmit(onSubmit)}
        className="flex flex-1 flex-col">
        <div className="flex items-center gap-3 border-b border-line px-4 py-3.5">
          <Glyph icon={glyph.icon} tone={glyph.tone} size="md" />
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="truncate text-[15px] font-semibold capitalize">
                {connector.name}
              </span>
              {isCode && (
                <span
                  className={cn(
                    "rounded-[5px] px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide",
                    isJsCode ? "bg-moss-soft text-moss-text" : "bg-signal-soft text-signal-text"
                  )}>
                  {isJsCode ? "node" : "python"}
                </span>
              )}
              {isCondition && (
                <span className="rounded-[5px] bg-slate-soft px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-slate-text">
                  go · in-worker
                </span>
              )}
            </div>
            <div className="text-[12.5px] text-ink-faint">Configure this step</div>
          </div>
        </div>

        <div className="flex flex-1 flex-col gap-4 overflow-auto p-4">
          <FormField
            control={taskForm.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Step name
                </FormLabel>
                <FormControl>
                  <Input placeholder="e.g. VirusTotal Lookup" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={taskForm.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Description
                  <span className="ml-1 font-medium normal-case text-ink-faint">— optional</span>
                </FormLabel>
                <FormControl>
                  <Textarea rows={2} placeholder="What this step does…" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {!isVirtual && connector.configs && (
            <FormField
              control={taskForm.control}
              name="config"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Configuration
                  </FormLabel>
                  <Select value={field.value ?? ""} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a saved configuration" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent className="bg-popover">
                      {connector.configs?.map((_config) => (
                        <SelectItem value={_config} key={`connector-config-${_config}`}>
                          {_config}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {/* condition: choose the case style with a mode toggle instead of a
              generic operation dropdown. code snippets imply their operation. */}
          {isCondition ? (
            <FormField
              control={taskForm.control}
              name="operation"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Cases — first match wins
                  </FormLabel>
                  <div className="inline-flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-1">
                    {connector.operations.map((op) => (
                      <button
                        key={op.annotation}
                        type="button"
                        onClick={() => field.onChange(op.annotation)}
                        className={cn(
                          "rounded-[6px] px-3 py-1.5 text-[12px] font-semibold transition-colors",
                          field.value === op.annotation
                            ? "bg-card text-ink shadow-sm"
                            : "text-ink-soft hover:text-ink"
                        )}>
                        {op.annotation === "switch"
                          ? "Simple compare"
                          : "Advanced expression"}
                      </button>
                    ))}
                  </div>
                  {currentOperation?.description && (
                    <FormDescription className="text-[11.5px] text-ink-faint">
                      {currentOperation.description}
                    </FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : !isVirtual ? (
            <FormField
              control={taskForm.control}
              name="operation"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Operation
                  </FormLabel>
                  <Select value={field.value ?? ""} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select an operation" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent className="bg-popover">
                      {connector.operations.map((operation) => (
                        <SelectItem
                          value={operation.annotation}
                          key={`connector-operation-${operation.title}`}>
                          {operation.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription className="text-[11.5px] text-ink-faint">
                    {currentOperation?.description}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}

          {/* parameters, inline (no separate tab) */}
          {operationName && currentOperation?.parameters?.length ? (
            <FormField
              control={taskForm.control}
              name="parameters"
              render={({ field }) => (
                <FormItem className="flex flex-col gap-4">
                  {currentOperation.parameters.map((param) => (
                    <div className="flex flex-col gap-1.5" key={`connector-operation-${param.name}`}>
                      {/* the case editors carry their own header; avoid a duplicate label */}
                      {param.type !== "cases" && param.type !== "conditions" && (
                        <>
                          <Label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                            {param.title}
                            {param.required && <span className="ml-1 text-rose-text">*</span>}
                          </Label>
                          {param.description && (
                            <p className="text-[11.5px] text-ink-faint">{param.description}</p>
                          )}
                        </>
                      )}
                      {param.type === "text" && (
                        <Input
                          placeholder={param.placeholder}
                          value={field.value?.[param.name] ?? ""}
                          onChange={(e) =>
                            field.onChange({
                              ...(field.value ? field.value : {}),
                              [param.name]: e.target.value,
                            })
                          }
                        />
                      )}
                      {param.type === "cases" && (
                        <CasesEditor
                          placeholder={param.placeholder}
                          value={field.value?.[param.name]}
                          onChange={(cases) =>
                            field.onChange({
                              ...(field.value ? field.value : {}),
                              [param.name]: cases,
                            })
                          }
                        />
                      )}
                      {param.type === "conditions" && (
                        <ConditionsEditor
                          placeholder={param.placeholder}
                          value={field.value?.[param.name]}
                          onChange={(conditions) =>
                            field.onChange({
                              ...(field.value ? field.value : {}),
                              [param.name]: conditions,
                            })
                          }
                        />
                      )}
                      {param.type === "code" && (
                        <div className="overflow-hidden rounded-md border border-line-strong">
                          <div className="flex items-center justify-between border-b border-line bg-paper-sunken px-2.5 py-1.5">
                            <span className="rounded-[5px] border border-line-strong bg-card px-1.5 py-0.5 font-mono text-[10.5px] font-semibold text-ink-soft">
                              {isJsCode ? "node" : "python3 -I"}
                            </span>
                            <span className="font-mono text-[10.5px] text-ink-faint">
                              stdout is the result
                            </span>
                          </div>
                          <CodeMirror
                            height="220px"
                            extensions={[isJsCode ? javascript() : python()]}
                            theme={resolvedTheme === "dark" ? "dark" : "light"}
                            value={field.value?.[param.name] ?? ""}
                            onChange={(val) =>
                              field.onChange({
                                ...(field.value ? field.value : {}),
                                [param.name]: val,
                              })
                            }
                          />
                        </div>
                      )}
                    </div>
                  ))}
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}
        </div>

        <footer className="mt-auto border-t border-line bg-card p-3">
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={closeSidebar}>
              Close
            </Button>
            <Button type="submit">Save step</Button>
          </div>
        </footer>
      </form>
    </Form>
  );
};

export default ConnectorOperation;
