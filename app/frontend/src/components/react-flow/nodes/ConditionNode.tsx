import { useEffect } from "react";
import {
  Handle,
  Node,
  NodeProps,
  Position,
  useStore,
  useUpdateNodeInternals,
} from "@xyflow/react";
import { GitBranch } from "lucide-react";
import { Avatar, AvatarFallback } from "../../ui/avatar";
import { Tasks } from "@/services/playbooks/playbooks.schema";
import {
  CONDITION_OUTPUT_HANDLE,
  FLOW_SELECT_TRIGGER_ID,
} from "@/settings/reactFlowIds";
import DeleteNodeButton from "./DeleteNodeButton";

// Routing is chosen in the node's form, which sets each outgoing edge's
// source_handle to a branch (a switch case id or "else"). React Flow only renders
// an edge if its source_handle matches a real handle, so we render a hidden handle
// for every branch the edges reference — otherwise an assigned edge detaches.
const ConditionNode: React.FC<NodeProps<Node<Tasks>>> = (props) => {
  const branchKey = useStore((s) => {
    const ids: string[] = [];
    s.edges.forEach((edge) => {
      if (
        edge.source === props.id &&
        edge.sourceHandle &&
        edge.sourceHandle !== CONDITION_OUTPUT_HANDLE
      )
        ids.push(edge.sourceHandle);
    });
    return ids.sort().join("|");
  });
  const branchHandles = branchKey ? branchKey.split("|") : [];

  // React Flow won't notice handles we add/remove on the fly, so re-measure
  // whenever the branch set changes — without this a newly routed edge can't
  // attach and stays disconnected.
  const updateNodeInternals = useUpdateNodeInternals();
  useEffect(() => {
    updateNodeInternals(props.id);
  }, [branchKey, props.id, updateNodeInternals]);

  return (
    <div className="p-3 group rounded-xl">
      <div className="flex items-center w-full gap-3">
        <Avatar className="size-10">
          <AvatarFallback>
            <GitBranch className="size-6 text-primary" />
          </AvatarFallback>
        </Avatar>
        <div className="flex flex-col items-start mr-11">
          <p className="text-lg font-medium">Condition</p>
          <p className="text-sm text-muted-foreground">{props.data.name}</p>
        </div>
      </div>

      {props.id != FLOW_SELECT_TRIGGER_ID && (
        <>
          <Handle type="target" position={Position.Top} id="target-top" className="!w-2 !h-2" />
          <Handle type="target" position={Position.Left} id="target-left" className="!w-2 !h-2" />
          <Handle type="target" position={Position.Right} id="target-right" className="!w-2 !h-2" />

          {/* the handle the user drags from */}
          <Handle
            type="source"
            position={Position.Bottom}
            id={CONDITION_OUTPUT_HANDLE}
            className="!w-3 !h-3"
          />

          {/* co-located hidden anchors so routed edges stay attached */}
          {branchHandles.map((id) => (
            <Handle
              key={id}
              type="source"
              position={Position.Bottom}
              id={id}
              isConnectable={false}
              style={{ opacity: 0 }}
            />
          ))}

          <DeleteNodeButton nodeId={props.id} />
        </>
      )}
    </div>
  );
};

export default ConditionNode;
