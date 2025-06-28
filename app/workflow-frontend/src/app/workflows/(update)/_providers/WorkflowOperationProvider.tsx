import React, {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  addEdge,
  Connection,
  Edge,
  FinalConnectionState,
  Node,
  OnEdgesChange,
  OnNodesChange,
  useEdgesState,
  useNodesState,
  useReactFlow,
} from "@xyflow/react";
import { PlaybookTaskNode } from "@/components/react-flow/schema";
import { useMutation, UseMutationResult, useQuery, UseQueryResult } from "@tanstack/react-query";
import {
  Edges,
  Tasks,
  UpdateWorkflowPayload,
  Workflow,
  WorkflowDataToUpdate,
} from "@/services/worfklows/workflows.schema";
import { FLOW_SELECT_TRIGGER_ID, FLOW_START_ID } from "@/settings/reactFlowIds";
import { ConnectorInfo } from "@/services/connectors/connectors.schema";
import ConnectorService from "@/services/connectors/connectors";
import WorkflowService from "@/services/worfklows/workflows";
import { toast } from "@/hooks/use-toast";

export type TaskOperationType =
  | "connector"
  | "utility"
  | "code"
  | "decision"
  | "wait"
  | "approval"
  | "input_prompt"
  | null;

export type WorkflowOperationType = {
  connectorQuery: UseQueryResult<ConnectorInfo[], Error> | null;
  updateWorkflowMutation: UseMutationResult<Workflow, Error, UpdateWorkflowPayload, unknown> | null
  connector: ConnectorInfo | null;
  setConnector: React.Dispatch<React.SetStateAction<ConnectorInfo | null>>;
  taskOperation: TaskOperationType;
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>;
  openOperationSidebar: boolean;
  setOpenOperationSidebar: React.Dispatch<React.SetStateAction<boolean>>;
  workflowData: WorkflowDataToUpdate;
  setWorkflowData: React.Dispatch<React.SetStateAction<WorkflowDataToUpdate>>;
  currentNode: Node<PlaybookTaskNode> | null;
  setCurrentNode: React.Dispatch<
    React.SetStateAction<Node<PlaybookTaskNode> | null>
  >;
  nodes: Node<PlaybookTaskNode>[];
  setNodes: React.Dispatch<React.SetStateAction<Node<PlaybookTaskNode>[]>>;
  onNodesChange: OnNodesChange<Node<PlaybookTaskNode>>;
  edges: Edge[];
  setEdges: React.Dispatch<React.SetStateAction<Edge[]>>;
  onEdgesChange: OnEdgesChange<Edge>;
  hasTriggerStep: boolean;
  onConnect: (params: Connection) => void;
  onConnectEnd: (
    event: MouseEvent | TouchEvent,
    connectionState: FinalConnectionState
  ) => void;
  isNewNode: boolean;
  setIsNewNode: React.Dispatch<React.SetStateAction<boolean>>;
  closeSidebar: () => void;
};

export const WorkflowOperationContext = createContext<WorkflowOperationType>({
  connectorQuery: null,
  connector: null,
  updateWorkflowMutation: null,
  setConnector: () => {},
  taskOperation: null,
  setTaskOperation: () => {},
  openOperationSidebar: false,
  setOpenOperationSidebar: () => {},
  workflowData: {},
  setWorkflowData: () => {},
  currentNode: null,
  setCurrentNode: () => {},
  nodes: [],
  setNodes: () => {},
  onNodesChange: () => {},
  edges: [],
  setEdges: () => {},
  onEdgesChange: () => {},
  hasTriggerStep: false,
  onConnect: (params: Connection) => {},
  onConnectEnd: (
    event: MouseEvent | TouchEvent,
    connectionState: FinalConnectionState
  ) => {},
  isNewNode: false,
  setIsNewNode: () => {},
  closeSidebar: () => {},
});

const INITIAL_START_NODE_VALUE: Node<PlaybookTaskNode> = {
  id: FLOW_SELECT_TRIGGER_ID,
  data: {
    label: FLOW_SELECT_TRIGGER_ID.replace("_", " "),
  },
  position: {
    x: 100,
    y: 100,
  },
  type: "start",
  draggable: false,
};

const WorkflowOperationProvider: React.FC<{
  children: any;
  workflowQuery: UseQueryResult<Workflow, Error>;
}> = ({ children, workflowQuery }) => {
  const connectorQuery = useQuery({
    queryKey: ["connectors-lists"],
    queryFn: async () => {
      return await ConnectorService.getConnectors();
    },
  });
  const updateWorkflowMutation = useMutation({
    mutationFn: async(workflow: UpdateWorkflowPayload) => {
      return await WorkflowService.updateWorkflow(workflowQuery.data?.id!, workflow)
    },
    onSuccess: (data) => {
      toast({
        title: "succesfully updated the workflow",
      })
    },
    onError(error) {
      toast({
        title: "Error when updating a new workflow",
        description: error.message,
      })
    },
  })
  const [connector, setConnector] = useState<ConnectorInfo | null>(null);
  const [taskOperation, setTaskOperation] = useState<TaskOperationType>(null); // this is to show what operation we need to show in the container
  const [openOperationSidebar, setOpenOperationSidebar] = useState(false);
  const [isNewNode, setIsNewNode] = useState(false);
  const [workflowData, setWorkflowData] = useState<WorkflowDataToUpdate>({});
  const [nodes, setNodes, onNodesChange] = useNodesState<
    Node<PlaybookTaskNode>
  >([]);
  const [currentNode, setCurrentNode] = useState<Node<PlaybookTaskNode> | null>(
    null
  );
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const { screenToFlowPosition } = useReactFlow();
  let id = 1;
  const getId = () => `${id++}`;

  useEffect(() => {
    if (workflowQuery.status == "error") {
      toast({
        title: "error fetching worfklow",
        description: workflowQuery.error.message
      })
      
    }

  }, [workflowQuery.status])

  useEffect(() => {
    const setMappedNodes = (task: Tasks) => {
      const data: Node<PlaybookTaskNode> = {
        id: task.id,
        data:
          task.name === FLOW_START_ID
            ? {
                label: "start",
                ...task,
              }
            : task,
        position: {
          x: task.x,
          y: task.y,
        },
        type: task.name === "start" ? "input" : "playbookNodes",
        draggable: true,
      };

      return data;
    };

    const setMappedEdges = (edge: Edges) => ({
      id: edge.id,
      source: edge.source_id,
      target: edge.destination_id,
    });
    const _nodes = workflowQuery.data?.tasks?.map(setMappedNodes) ?? [];
    // if task doesnt have a node with a name start,
    // add a new node for selecting a trigger. open the sidebar
    // operation to notify the user to select trigger quickly
    if (_nodes.some((task) => task.data?.name === FLOW_START_ID) == false) {
      _nodes.unshift(INITIAL_START_NODE_VALUE);
      setCurrentNode(INITIAL_START_NODE_VALUE);
      setOpenOperationSidebar(true);
    }
    setNodes(_nodes);
    setEdges(workflowQuery.data?.edges?.map(setMappedEdges) ?? []);
    if (workflowQuery.data)
      setWorkflowData({
        name: workflowQuery.data.name,
        description: workflowQuery.data.description,
        trigger_type: workflowQuery.data.trigger_type,
      });
  }, [workflowQuery.isFetched]);

  /**
   * return false if no tasks has a name start
   * else true. We need this to know if we have to show
   * options for the workflow trigger or
   * to select task options
   */
  const hasTriggerStep = useMemo(() => {
    return (
      workflowData.trigger_type != undefined ||
      workflowData.trigger_type != null
    );
  }, [workflowData]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    []
  );
  const onConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent, connectionState: FinalConnectionState) => {
      // when a connection is dropped on the pane it's not valid
      if (!connectionState.isValid) {
        // we need to remove the wrapper bounds, in order to get the correct position
        const id = getId();
        const { clientX, clientY } =
          "changedTouches" in event ? event.changedTouches[0] : event;
        const newNode: Node<PlaybookTaskNode> = {
          id,
          position: screenToFlowPosition({
            x: clientX,
            y: clientY,
          }),
          data: {},
          type: "playbookNodes",
          draggable: true,
        };

        setNodes((nds) => nds.concat(newNode));
        setCurrentNode(newNode);

        setEdges((eds) =>
          eds.concat({
            id,
            source: connectionState.fromNode!.id,
            target: id,
          })
        );

        setOpenOperationSidebar(true);
        setIsNewNode(true);
      }
    },
    [screenToFlowPosition]
  );

  const closeSidebar = () => {
    setOpenOperationSidebar(false);
    setCurrentNode(null);
    setIsNewNode(false);
    setTaskOperation(null);
  };
  return (
    <WorkflowOperationContext.Provider
      value={{
        connectorQuery,
        updateWorkflowMutation,
        connector,
        setConnector,
        taskOperation,
        setTaskOperation,
        openOperationSidebar,
        setOpenOperationSidebar,
        workflowData,
        setWorkflowData,
        nodes,
        currentNode,
        setCurrentNode,
        setNodes,
        onNodesChange,
        edges,
        setEdges,
        onEdgesChange,
        hasTriggerStep,
        onConnect,
        onConnectEnd,
        isNewNode,
        setIsNewNode,
        closeSidebar,
      }}>
      {children}
    </WorkflowOperationContext.Provider>
  );
};

export default WorkflowOperationProvider;
