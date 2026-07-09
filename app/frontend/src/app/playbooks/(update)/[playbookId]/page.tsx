"use client";

import React, { useContext } from "react";

import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";
import { PlaybookTaskNode } from "@/components/react-flow/schema";
import { Node, ReactFlowProvider } from "@xyflow/react";

import { useQuery } from "@tanstack/react-query";
import PlaybookService from "@/services/playbooks/playbooks";
import { Button } from "@/components/ui/button";
import {
  UpdateHandlesPayload,
  UpdatePlaybookPayload,
} from "@/services/playbooks/playbooks.schema";
import PlaybookOperationProvider, {
  PlaybookOperationContext,
} from "../_providers/PlaybookOperationProvider";
import PlaybookOperations from "../_components/options/PlaybookOperations/PlaybookOperations";
import usePlaybookTrigger from "@/hooks/usePlaybookTrigger";
import { FLOW_START_ID } from "@/settings/reactFlowIds";
import { Check, History, Workflow as PlaybookIcon, Zap } from "lucide-react";
import Link from "next/link";

// Canvas nav: a segmented Editor / Runs toggle pinned to the top-left of the
// canvas. Replaces the two stacked bordered buttons that used to float mid-left.
const RouterButton: React.FC<{ playbookId: string }> = ({ playbookId }) => {
  return (
    <div className="absolute left-4 top-4 z-10 inline-flex items-center gap-1 rounded-md border border-line bg-card p-1 shadow-md">
      <span className="inline-flex items-center gap-1.5 rounded-sm bg-signal-soft px-3 py-1.5 text-[12.5px] font-semibold text-signal-text">
        <PlaybookIcon className="size-3.5" /> Editor
      </span>
      <Link
        href={"/playbooks/" + playbookId + "/history"}
        className="inline-flex items-center gap-1.5 rounded-sm px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft transition-colors hover:bg-paper-sunken hover:text-ink"
      >
        <History className="size-3.5" /> Runs
      </Link>
    </div>
  );
};

const PlaybookPlayground: React.FC<{ playbookId: string }> = ({
  playbookId,
}) => {
  const {
    nodes,
    onNodesChange,
    onConnect,
    onConnectEnd,
    edges,
    onEdgesChange,
    setCurrentNode,
    setOpenOperationSidebar,
    setConnector,
    connectorQuery,
    setTaskOperation,
    openOperationSidebar,
    workflowData,
    updatePlaybookMutation,
    onNodesDelete,
  } = useContext(PlaybookOperationContext);

  const { triggerPlaybookHandler } = usePlaybookTrigger({ playbookId });

  // set the connector to the node's connector
  // can improve later
  const setConnectorToNodesConnector = (node: Node<PlaybookTaskNode>) => {
    if (connectorQuery && node.data.connector_id) {
      for (const _connector of connectorQuery?.data || []) {
        if (_connector.id == node.data.connector_id) {
          setConnector(_connector || null);
          setTaskOperation("connector");
          break;
        }
      }
    }
  };

  /**
   * open widget for updating node
   * @param e
   * @param node
   */
  const onNodeDoubleClick = (
    e: React.MouseEvent<Element, MouseEvent>,
    node: Node<PlaybookTaskNode>
  ) => {
    setOpenOperationSidebar(true);
    if (node.id !== "select_start") {
      setCurrentNode(node);
      setConnectorToNodesConnector(node);
    } else if (node.data.name == FLOW_START_ID) {
    }
  };

  /**
   * save all the updated workflow
   */
  const savePlaybookHandler = () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const data: Record<string, any> = {};
    const node_mapper: Record<string, string> = nodes.reduce(
      (prev, curr) => ({ ...prev, [curr.id!]: curr.data.name }),
      {}
    );
    data.task = {
      name: workflowData.name,
      trigger_type: workflowData.trigger_type,
    };

    data.nodes = nodes.map((_node) => ({
      ..._node.data,
      x: _node.position.x,
      y: _node.position.y,
    }));

    // create the edges
    data["edges"] = edges.reduce((prev, curr) => {
      const sourceNodeName = node_mapper[curr.source];
      const destNodeName = node_mapper[curr.target];
      if (sourceNodeName in prev) {
        prev[sourceNodeName].push(destNodeName);
      } else {
        prev[sourceNodeName] = [destNodeName];
      }
      return prev;
    }, {} as Record<string, string[]>);

    // create the handles
    data["handles"] = edges.reduce((prev, curr) => {
      const sourceNodeName = node_mapper[curr.source];
      const destNodeName = node_mapper[curr.target];
      if (sourceNodeName in prev) {
        prev[sourceNodeName][destNodeName] = {
          source_handle: curr.sourceHandle,
          destination_handle: curr.targetHandle,
        };
      } else {
        prev[sourceNodeName] = {
          [destNodeName]: {
            source_handle: curr.sourceHandle || null,
            destination_handle: curr.targetHandle || null,
          },
        };
      }
      return prev;
    }, {} as UpdateHandlesPayload);

    if (updatePlaybookMutation)
      updatePlaybookMutation.mutate(data as UpdatePlaybookPayload);
  };

  return (
    <div className="relative h-[calc(100vh-4rem)]">
      {/* <div className='absolute left-0 z-50 flex flex-col border-r bg-background border-r-border top-40'>
        <Button className="rounded-e-2xl" onClick={() => setOpenOperationSidebar(true)}>
          <ArrowRightIcon />
        </Button>
      </div> */}
      {openOperationSidebar && <PlaybookOperations />}

      <div className="flex h-16 items-center justify-between border-b border-line px-5 py-3">
        <div>
          <h2 className="text-lg font-semibold">{workflowData.name}</h2>
          <p className="text-xs text-ink-faint">{workflowData.description}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={triggerPlaybookHandler}>
            <Zap /> Trigger
          </Button>
          <Button variant="outline" className="text-rose-text">
            Delete
          </Button>
          <Button onClick={savePlaybookHandler}>
            <Check /> Save
          </Button>
        </div>
      </div>
      <div className="h-[calc(100vh-8rem)] relative">
        <RouterButton playbookId={playbookId} />
        <ReactFlowPlayground<PlaybookTaskNode>
          flowProps={{
            nodes,
            edges,
            onNodeDoubleClick,
            onNodesChange,
            onEdgesChange,
            onConnect,
            onConnectEnd,
            onNodesDelete,
          }}
        />
      </div>
    </div>
  );
};

const Page: React.FC<{ params: Promise<{ playbookId: string }> }> = ({
  params,
}) => {
  const { playbookId } = React.use(params);
  console.log(playbookId)

  const workflowQuery = useQuery({
    queryKey: ["workflow-" + playbookId, playbookId],
    queryFn: async () => {
      return PlaybookService.getPlaybookById(playbookId);
    },
  });

  if (workflowQuery.isLoading) {
    return null;
  }

  if (workflowQuery.isError) {
    return <div>No workflow found</div>;
  }

  return (
    <ReactFlowProvider>
      <PlaybookOperationProvider workflowQuery={workflowQuery}>
        <PlaybookPlayground playbookId={playbookId} />
      </PlaybookOperationProvider>
    </ReactFlowProvider>
  );
};

export default Page;
