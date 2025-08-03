"use client";

import React, { useContext } from "react";

import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";
import { PlaybookTaskNode } from "@/components/react-flow/schema";
import { Node, ReactFlowProvider } from "@xyflow/react";

import { useQuery } from "@tanstack/react-query";
import WorkflowService from "@/services/worfklows/workflows";
import { Button } from "@/components/ui/button";
import {
  UpdateHandlesPayload,
  UpdateWorkflowPayload,
} from "@/services/worfklows/workflows.schema";
import WorkflowOperationProvider, {
  WorkflowOperationContext,
} from "../../_providers/WorkflowOperationProvider";
import WorkflowOperations from "../../_components/options/WorkflowOperations/WorkflowOperations";
import useWorkflowTrigger from "@/hooks/useWorkflowTrigger";
import { FLOW_START_ID } from "@/settings/reactFlowIds";

const Page: React.FC<{ params: Promise<{ workflow_id: string }> }> = ({
  params,
}) => {
  const { workflow_id: workflowId } = React.use(params);

  const workflowQuery = useQuery({
    queryKey: ["workflow-" + workflowId, workflowId],
    queryFn: async () => {
      return WorkflowService.getWorkflowById(workflowId);
    },
    
  });

  if (workflowQuery.isLoading) {
    return null;
  }

  if(workflowQuery.isError) {
    return <div>No workflow found</div>
  }

  return (
    <ReactFlowProvider>
      <WorkflowOperationProvider workflowQuery={workflowQuery}>
        <WorkflowPlayground workflowId={workflowId} />
      </WorkflowOperationProvider>
    </ReactFlowProvider>
  );
};

const WorkflowPlayground: React.FC<{ workflowId: string }> = ({
  workflowId,
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
    updateWorkflowMutation,
    onNodesDelete
  } = useContext(WorkflowOperationContext);

  const { triggerWorkflowHandler } = useWorkflowTrigger({ workflowId });

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
    console.log(node);
  };

  /**
   * save all the updated workflow
   */
  const saveWorkflowHandler = () => {
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

    if (updateWorkflowMutation)
      updateWorkflowMutation.mutate(data as UpdateWorkflowPayload);
  };

  return (
    <div className="relative h-[calc(100vh-4rem)]">
      {/* <div className='absolute left-0 z-50 flex flex-col border-r bg-background border-r-border top-40'>
        <Button className="rounded-e-2xl" onClick={() => setOpenOperationSidebar(true)}>
          <ArrowRightIcon />
        </Button>
      </div> */}
      {openOperationSidebar && <WorkflowOperations />}

      <div className="flex items-center justify-between h-16 px-5 py-3">
        <div>
          <h2 className="text-xl font-medium">{workflowData.name}</h2>
          <p className="text-xs text-muted-foreground">
            {workflowData.description}asd asd asd
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={triggerWorkflowHandler}>Trigger</Button>
          <Button>Delete</Button>
          <Button onClick={saveWorkflowHandler}>Save</Button>
        </div>
      </div>
      <div className="h-[calc(100vh-8rem)]">
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

export default Page;
