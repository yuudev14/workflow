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
import {
  useMutation,
  UseMutationResult,
  useQuery,
  UseQueryResult,
} from "@tanstack/react-query";
import {
  Edges,
  Tasks,
  UpdatePlaybookPayload,
  Playbook,
  PlaybookDataToUpdate,
} from "@/services/playbooks/playbooks.schema";
import {
  CONDITION_CONNECTOR_ID,
  FLOW_SELECT_TRIGGER_ID,
  FLOW_START_ID,
} from "@/settings/reactFlowIds";
import { ConnectorInfo } from "@/services/connectors/connectors.schema";
import ConnectorService from "@/services/connectors/connectors";
import PlaybookService from "@/services/playbooks/playbooks";
import { toast } from "@/hooks/use-toast";

export type TaskOperationType =
  | "connector"
  | "utility"
  | "code"
  | "code_js"
  | "decision"
  | "wait"
  | "approval"
  | "input_prompt"
  | null;

export type PlaybookOperationType = {
  connectorQuery: UseQueryResult<ConnectorInfo[], Error> | null;
  updatePlaybookMutation: UseMutationResult<
    Playbook,
    Error,
    UpdatePlaybookPayload,
    unknown
  > | null;
  connector: ConnectorInfo | null;
  setConnector: React.Dispatch<React.SetStateAction<ConnectorInfo | null>>;
  taskOperation: TaskOperationType;
  setTaskOperation: React.Dispatch<React.SetStateAction<TaskOperationType>>;
  openOperationSidebar: boolean;
  setOpenOperationSidebar: React.Dispatch<React.SetStateAction<boolean>>;
  workflowData: PlaybookDataToUpdate;
  setPlaybookData: React.Dispatch<React.SetStateAction<PlaybookDataToUpdate>>;
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
  onNodesDelete: (node: Node<PlaybookTaskNode>[]) => void;
};

export const PlaybookOperationContext = createContext<PlaybookOperationType>({
  connectorQuery: null,
  connector: null,
  updatePlaybookMutation: null,
  setConnector: () => {},
  taskOperation: null,
  setTaskOperation: () => {},
  openOperationSidebar: false,
  setOpenOperationSidebar: () => {},
  workflowData: {},
  setPlaybookData: () => {},
  currentNode: null,
  setCurrentNode: () => {},
  nodes: [],
  setNodes: () => {},
  onNodesChange: () => {},
  edges: [],
  setEdges: () => {},
  onEdgesChange: () => {},
  hasTriggerStep: false,
  onConnect: (_params: Connection) => {},
  onConnectEnd: (
    _event: MouseEvent | TouchEvent,
    _connectionState: FinalConnectionState
  ) => {},
  isNewNode: false,
  setIsNewNode: () => {},
  closeSidebar: () => {},
  onNodesDelete: (_node: Node<PlaybookTaskNode>[]) => {},
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
  type: "playbookNodes",
  draggable: true,
};

const PlaybookOperationProvider: React.FC<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  children: any;
  workflowQuery: UseQueryResult<Playbook, Error>;
}> = ({ children, workflowQuery }) => {
  const connectorQuery = useQuery({
    queryKey: ["connectors-lists"],
    queryFn: async () => {
      return await ConnectorService.getConnectors();
    },
  });
  const updatePlaybookMutation = useMutation({
    mutationFn: async (workflow: UpdatePlaybookPayload) => {
      return await PlaybookService.updatePlaybook(
        workflowQuery.data!.id!,
        workflow
      );
    },
    onSuccess: (_data) => {
      toast({
        variant: "success",
        title: "Playbook saved",
        description: "Your changes have been saved.",
      });
    },
    onError(error) {
      toast({
        variant: "destructive",
        title: "Couldn't save the playbook",
        description: error.message,
      });
    },
  });
  const [connector, setConnector] = useState<ConnectorInfo | null>(null);
  const [taskOperation, setTaskOperation] = useState<TaskOperationType>(null); // this is to show what operation we need to show in the container
  const [openOperationSidebar, setOpenOperationSidebar] = useState(false);
  const [isNewNode, setIsNewNode] = useState(false);
  const [workflowData, setPlaybookData] = useState<PlaybookDataToUpdate>({});
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
        variant: "destructive",
        title: "Couldn't load the playbook",
        description: workflowQuery.error.message,
      });
    }
  }, [workflowQuery.status]);

  useEffect(() => {
    const setMappedNodes = (task: Tasks) => {
      const data: Node<PlaybookTaskNode> = {
        id: task.id,
        data:
          task.name === FLOW_START_ID
            ? {
                label: task.id,
                ...task,
              }
            : task,
        position: {
          x: task.x,
          y: task.y,
        },
        type:
          task.name === FLOW_START_ID
            ? "startNode"
            : task.connector_id === CONDITION_CONNECTOR_ID
            ? "conditionNode"
            : "playbookNodes",
        draggable: true,
      };

      return data;
    };

    const setMappedEdges = (edge: Edges) => ({
      id: edge.id,
      source: edge.source_id,
      target: edge.destination_id,
      sourceHandle: edge.source_handle || "source-top",
      targetHandle: edge.destination_handle || "target-top",
      type: "removableEdge",
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
      setPlaybookData({
        name: workflowQuery.data.name,
        description: workflowQuery.data.description,
        trigger_type: workflowQuery.data.trigger_type,
        trigger_parameters: workflowQuery.data.trigger_parameters,
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

  const onNodesDelete = useCallback((node: Node<PlaybookTaskNode>[]) => {
    console.log(node);
    if (node.find((_node) => _node.data.name == FLOW_START_ID)) {
      setPlaybookData((data) => ({ ...data, trigger_type: null, trigger_parameters: null }));
      setNodes((_nodes) => _nodes.concat(INITIAL_START_NODE_VALUE));
      setCurrentNode(INITIAL_START_NODE_VALUE);
      setOpenOperationSidebar(true);
    }
  }, []);

  const onConnect = useCallback((params: Connection) => {
    setEdges((eds) => addEdge(params, eds));
  }, []);
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
            // keep the handle the user actually dragged from — without this the
            // edge falls back to the node's first source handle (source-top),
            // so every new-step wire wrongly exits the top of the source node.
            sourceHandle: connectionState.fromHandle?.id ?? undefined,
            target: id,
            targetHandle: "target-left",
          })
        );

        setOpenOperationSidebar(true);
        setIsNewNode(true);
      }
    },
    [screenToFlowPosition]
  );

  const closeSidebar = () => {
    // remove filters if currennode exist
    setOpenOperationSidebar(false);
    setCurrentNode(null);
    setIsNewNode(false);
    setTaskOperation(null);
    setConnector(null);
  };
  return (
    <PlaybookOperationContext.Provider
      value={{
        connectorQuery,
        updatePlaybookMutation,
        connector,
        setConnector,
        taskOperation,
        setTaskOperation,
        openOperationSidebar,
        setOpenOperationSidebar,
        workflowData,
        setPlaybookData,
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
        onNodesDelete,
      }}>
      {children}
    </PlaybookOperationContext.Provider>
  );
};

export default PlaybookOperationProvider;
