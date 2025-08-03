import settings from "@/settings";
import { EntryResponse } from "../common/schema";
import {
  CreateWorkflowPayload,
  UpdateWorkflowPayload,
  Workflow,
  WorkflowFilterPayload,
  WorkflowHistory,
  WorkflowTriggerType,
} from "./workflows.schema";
import apiClient from "../common/client";

export default class WorkflowService {
  private static BASE_URL =
    settings.BASE_URL.WORKFLOW_SERVICE_API + "/api/workflows/v1";
  /**
   * get workflows lists
   * @param offset
   * @param limit
   * @param worfklowFilter
   * @returns an entry response ex:
   * {
   *  "entries": [],
   *  "total": 0
   * }
   */
  public static getWorkflows = async (
    offset: number = 0,
    limit: number = 50,
    worfklowFilter: WorkflowFilterPayload = {}
  ): Promise<EntryResponse<Workflow>> => {
    const res = await apiClient.get(
      this.BASE_URL,
      {
        params: {
          offset,
          limit,
          ...worfklowFilter,
        },
      }
    );
    return res.data;
  };


  /**
   * get workflows history
   * @param offset
   * @param limit
   * @returns an entry response ex:
   * {
   *  "entries": [],
   *  "total": 0
   * }
   */
  public static getWorkflowsHistory = async (
    offset: number = 0,
    limit: number = 50,
  ): Promise<EntryResponse<WorkflowHistory>> => {
    const res = await apiClient.get(
      this.BASE_URL + "/history",
      {
        params: {
          offset,
          limit,
        },
      }
    );
    return res.data;
  };

  /**
   * get workflow trigger types
   * @returns an entry response ex:
   * {
   *  "entries": [],
   *  "total": 0
   * }
   */
  public static getWorkflowTriggerTypes = async (): Promise<
    WorkflowTriggerType[]
  > => {
    const res = await apiClient.get(
      this.BASE_URL + "/triggers"
    );
    return res.data;
  };

  /**
   * get workflows by id
   * @param workflowId
   * @returns worfklow object
   */
  public static getWorkflowById = async (
    workflowId: string
  ): Promise<Workflow> => {
    const res = await apiClient.get(
      this.BASE_URL + "/" + workflowId
    );
    return res.data;
  };


  /**
   * get workflows by id
   * @param workflowId
   * @returns worfklow object
   */
  public static triggerWorkflow = async (
    workflowId: string
  ): Promise<Workflow> => {
    const res = await apiClient.post(
      this.BASE_URL + "/trigger/" + workflowId
    );
    return res.data;
  };

  /**
   * create worflow api
   * @param payload
   * @returns
   */
  public static createWorkflow = async (
    payload: CreateWorkflowPayload
  ): Promise<Workflow> => {
    const res = await apiClient.post(
      this.BASE_URL,
      payload
    );
    return res.data;
  };

  /**
   * update workflow api
   * @param workflowId 
   * @param payload 
   * @returns 
   */
  public static updateWorkflow = async (
    workflowId: string,
    payload: UpdateWorkflowPayload
  ): Promise<Workflow> => {
    const res = await apiClient.put(
      this.BASE_URL + "/tasks/" + workflowId ,
      payload
    );
    return res.data;
  };
}
