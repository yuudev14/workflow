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
import { WorkflowOperationContext } from "../../../_providers/WorkflowOperationProvider";
import { Textarea } from "@/components/ui/textarea";
import Editor from "@monaco-editor/react";

const taskFormSchema = z.object({
  name: z.string().min(2, {
    message: "name must be at least 2 characters.",
  }),
  description: z.string().optional(),
  parameters: z.record(z.string()).nullable().optional(),
  config: z.string().nullable().optional(),
  connector_name: z.string(),
  connector_id: z.string(),
  operation: z.string(),
});

const ConnectorOperation: React.FC<{ connector: ConnectorInfo }> = ({
  connector,
}) => {
  const [currentOperation, setCurrentOperation] = useState<Operation | null>(
    null
  );
  const { currentNode, setNodes, closeSidebar } = useContext(
    WorkflowOperationContext
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
                        {param.type === "code" && (
                          <div className="flex-1 flex-col">
                            <Editor
                              height="100%"
                              language="python"
                              theme="vs-dark"
                              value={field.value?.[param.name] ?? ""}
                              onChange={(val) =>
                                field.onChange({
                                  ...(field.value ? field.value : {}),
                                  [param.name]: val,
                                })
                              }
                              options={{
                                automaticLayout: true,
                                fontSize: 14,
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
