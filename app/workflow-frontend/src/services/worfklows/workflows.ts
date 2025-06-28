import settings from "@/settings";
import { EntryResponse } from "../common/schema";
import {
  CreateWorkflowPayload,
  UpdateWorkflowPayload,
  Workflow,
  WorkflowFilterPayload,
  WorkflowTriggerType,
} from "./workflows.schema";
import apiClient from "../common/client";

export default class WorkflowService {
  private static BASE_URL =
    settings.BASE_URL.WORKFLOW_SERVICE_API;

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
      this.BASE_URL + "/api/v1/workflows",
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
      this.BASE_URL + "/api/v1/workflows/triggers"
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
      this.BASE_URL + "/api/v1/workflows/" + workflowId
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
      this.BASE_URL + "/api/v1/workflows",
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
      this.BASE_URL + "/api/v1/workflows/tasks/" + workflowId ,
      payload
    );
    return res.data;
  };
}
