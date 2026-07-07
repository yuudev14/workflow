import { useContext, useEffect, useRef, useState } from "react";
import {
  ConnectorInfo,
  Operation,
} from "@/services/connectors/connectors.schema";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
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
  CONDITION_CONNECTOR_ID,
  CONDITION_OUTPUT_HANDLE,
} from "@/settings/reactFlowIds";

import CodeMirror from '@uiw/react-codemirror';
import { python } from '@codemirror/lang-python';

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

// Dropdown that routes one condition branch to a destination. Options are the
// nodes already wired to this condition node; picking one stamps that branch's
// handle onto the edge so the executor follows it for that branch.
const BranchRouter: React.FC<{ handleId: string; label: string }> = ({
  handleId,
  label,
}) => {
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

  return (
    <div className="flex items-center gap-2">
      <span className="w-20 shrink-0 text-xs text-muted-foreground">{label}</span>
      {outgoing.length === 0 ? (
        <p className="text-xs text-muted-foreground">
          connect this node to a destination first
        </p>
      ) : (
        <Select value={selected?.target ?? ""} onValueChange={assign}>
          <SelectTrigger className="flex-1">
            <SelectValue placeholder="select destination" />
          </SelectTrigger>
          <SelectContent className="bg-background">
            <SelectItem value={UNASSIGN}>— none —</SelectItem>
            {outgoing.map((edge) => (
              <SelectItem value={edge.target} key={`route-${edge.id}`}>
                {nodeName(edge.target)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </div>
  );
};

// One switch branch: a truthy expression plus a stable id naming its edge handle.
type Case = { id: string; expression: string };

// Editor for the "switch" operation: an ordered list of if / else-if expressions,
// each routed to its own destination, plus a trailing else. Cases are evaluated
// top-to-bottom and the first truthy one wins.
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
    <div className="flex flex-col gap-4">
      {cases.map((c, i) => (
        <div className="flex flex-col gap-1" key={c.id}>
          <Input
            placeholder={placeholder}
            value={c.expression ?? ""}
            onChange={(e) => update(i, e.target.value)}
          />
          <div className="flex items-center gap-2">
            <BranchRouter handleId={c.id} label={i === 0 ? "if →" : "else if →"} />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => remove(i)}>
              Remove
            </Button>
          </div>
        </div>
      ))}
      <BranchRouter handleId="else" label="else →" />
      <Button type="button" variant="outline" size="sm" onClick={add}>
        Add case
      </Button>
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

  return (
    <div className="flex flex-col gap-4">
      {conditions.map((c, i) => (
        <div className="flex flex-col gap-1" key={c.id}>
          <div className="flex items-center gap-1">
            <Input
              placeholder={placeholder}
              value={c.left ?? ""}
              onChange={(e) => update(i, { left: e.target.value })}
            />
            <Select
              value={c.operator}
              onValueChange={(op) => update(i, { operator: op })}>
              <SelectTrigger className="w-36 shrink-0">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="bg-background">
                {OPERATORS.map((op) => (
                  <SelectItem key={op} value={op}>
                    {op}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              placeholder="value"
              value={c.right ?? ""}
              onChange={(e) => update(i, { right: e.target.value })}
            />
          </div>
          <div className="flex items-center gap-2">
            <BranchRouter handleId={c.id} label={i === 0 ? "if →" : "else if →"} />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => remove(i)}>
              Remove
            </Button>
          </div>
        </div>
      ))}
      <BranchRouter handleId="else" label="else →" />
      <Button type="button" variant="outline" size="sm" onClick={add}>
        Add condition
      </Button>
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [cachedParameter, setCachedParameter] = useState<Record<string, any>>(
    {}
  );

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
        className="flex flex-col flex-1">
        <div className="flex flex-col flex-1 h-full gap-3 p-3">
          <FormField
            control={taskForm.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Step Name</FormLabel>
                <FormControl>
                  <Input placeholder="task name" {...field} />
                </FormControl>
                <FormDescription />
                <FormMessage />
              </FormItem>
            )}
          />

          <Separator className="bg-secondary" />

          <FormField
            control={taskForm.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                  <Textarea cols={5} placeholder="description" {...field} />
                </FormControl>
                <FormDescription />
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="py-7">
            <p>connector information</p>
          </div>
          <Separator className="bg-secondary" />
          <div className="flex flex-col gap-3 flex-1">
            {connector.configs && (
              <FormField
                control={taskForm.control}
                name="config"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Configuration</FormLabel>
                    <Select
                      value={field.value ?? ""}
                      onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="select configuration" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent className="bg-background">
                        {connector.configs?.map((_config) => (
                          <SelectItem
                            value={_config}
                            key={`connector-config-${_config}`}>
                            {_config}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription />
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <Separator className="bg-secondary" />

            <FormField
              control={taskForm.control}
              name="operation"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Operation</FormLabel>
                  <Select
                    value={field.value ?? ""}
                    onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="select operation" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent className="bg-background">
                      {connector.operations.map((operation) => (
                        <SelectItem
                          value={operation.annotation}
                          key={`connector-operation-${operation.title}`}>
                          {operation.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription />
                  <FormMessage />
                </FormItem>
              )}
            />

            <Separator className="bg-secondary" />

            {currentOperation && currentOperation.parameters && (
              <FormField
                control={taskForm.control}
                name="parameters"
                render={({ field }) => (
                  <FormItem className="flex-1 flex flex-col">
                    <FormLabel>Parameters</FormLabel>
                    {currentOperation.parameters.map((param) => (
                      <div
                        className="flex flex-col gap-2 flex-1"
                        key={`connector-operation-${param.name}`}>
                        <Label className="font-normal">{param.title}</Label>
                        {param.type === "text" && (
                          <Input
                            placeholder={param.placeholder}
                            value={field.value?.[param.name] ?? ""}
                            onChange={(e) => {
                              field.onChange({
                                ...(field.value ? field.value : {}),
                                [param.name]: e.target.value,
                              });
                            }}
                          />
                        )}
                        {param.type === "cases" && (
                          <CasesEditor
                            placeholder={param.placeholder}
                            value={field.value?.[param.name]}
                            onChange={(cases) => {
                              field.onChange({
                                ...(field.value ? field.value : {}),
                                [param.name]: cases,
                              });
                            }}
                          />
                        )}
                        {param.type === "conditions" && (
                          <ConditionsEditor
                            placeholder={param.placeholder}
                            value={field.value?.[param.name]}
                            onChange={(conditions) => {
                              field.onChange({
                                ...(field.value ? field.value : {}),
                                [param.name]: conditions,
                              });
                            }}
                          />
                        )}
                        {param.type === "code" && (
                          <div className="flex-1 flex-col">
                            {/* <Editor
                              height="300px"
                              language="python"
                              theme="vs-dark"
                              // value={field.value?.[param.name] ?? ""}
                              onChange={(val) => {
                                console.log(val + "yuuu");
                                // field.onChange({
                                //   ...(field.value ? field.value : {}),
                                //   [param.name]: val,
                                // })
                              }}
                              options={{
                                automaticLayout: true,
                                fontSize: 14,

                                // Fix space/enter issue:
                                tabSize: 2,
                                insertSpaces: true,
                                detectIndentation: false,
                                cursorWidth: 2,
                                scrollBeyondLastLine: false,
                                // Ensure normal typing behavior
                                acceptSuggestionOnEnter: "on",
                                acceptSuggestionOnCommitCharacter: true,
                                quickSuggestions: false,
                              }}
                            /> */}
                            <CodeMirror 
                              height="200px"
                              extensions={[python()]}
                              theme={"dark"}
                              value={field.value?.[param.name] ?? ""}
                              onChange={(val) => {
                                field.onChange({
                                  ...(field.value ? field.value : {}),
                                  [param.name]: val,
                                })
                              }}
                            />
                          </div>
                        )}
                      </div>
                    ))}
                    <FormDescription />
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </div>
        </div>
        <footer className="p-3 mt-auto border border-t border-border">
          <div className="flex justify-between gap-2">
            <Button type="button">Close</Button>
            <Button>Save</Button>
          </div>
        </footer>
      </form>
      
    </Form>
  );
};

export default ConnectorOperation;
