import settings from "@/settings";
import { EntryResponse } from "../common/schema";
import {
  CreatePlaybookPayload,
  Edges,
  TaskHistory,
  UpdatePlaybookPayload,
  Playbook,
  PlaybookFilterPayload,
  PlaybookHistory,
  PlaybookHistoryFilter,
} from "./playbooks.schema";
import apiClient from "../common/client";

export default class PlaybookService {
  private static BASE_URL =
    settings.BASE_URL.PLAYBOOK_SERVICE_API + "/api/playbooks/v1";
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
  public static getPlaybooks = async (
    offset: number = 0,
    limit: number = 50,
    worfklowFilter: PlaybookFilterPayload = {}
  ): Promise<EntryResponse<Playbook>> => {
    const res = await apiClient.get(this.BASE_URL, {
      params: {
        offset,
        limit,
        ...worfklowFilter,
      },
    });
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
  public static getPlaybooksHistory = async (
    offset: number = 0,
    limit: number = 50,
    filter: PlaybookHistoryFilter = {}
  ): Promise<EntryResponse<PlaybookHistory>> => {
    const res = await apiClient.get(this.BASE_URL + "/history", {
      params: {
        offset,
        limit,
        ...filter
      },
    });
    return res.data;
  };

  public static getPlaybooksHistoryByPlaybookId = async (
    playbookId: string,
    offset: number = 0,
    limit: number = 50
  ): Promise<EntryResponse<PlaybookHistory>> => {
    return await this.getPlaybooksHistory(offset, limit, {playbook_id: playbookId});
  };


  public static getTaskHistoryByPlaybookHistoryId = async (
    worfklowHistoryId: string,
  ): Promise<{
    tasks: TaskHistory[];
    edges: Edges[];
  }> => {
    const res = await apiClient.get(`${this.BASE_URL}/history/${worfklowHistoryId}/tasks`);
    return res.data;
  };

  /**
   * get workflows by id
   * @param playbookId
   * @returns worfklow object
   */
  public static getPlaybookById = async (
    playbookId: string
  ): Promise<Playbook> => {
    const res = await apiClient.get(this.BASE_URL + "/" + playbookId);
    return res.data;
  };

  /**
   * get workflows by id
   * @param playbookId
   * @returns worfklow object
   */
  public static triggerPlaybook = async (
    playbookId: string
  ): Promise<Playbook> => {
    const res = await apiClient.post(this.BASE_URL + "/trigger/" + playbookId);
    return res.data;
  };

  /**
   * create worflow api
   * @param payload
   * @returns
   */
  public static createPlaybook = async (
    payload: CreatePlaybookPayload
  ): Promise<Playbook> => {
    const res = await apiClient.post(this.BASE_URL, payload);
    return res.data;
  };

  /**
   * update workflow api
   * @param playbookId
   * @param payload
   * @returns
   */
  public static updatePlaybook = async (
    playbookId: string,
    payload: UpdatePlaybookPayload
  ): Promise<Playbook> => {
    const res = await apiClient.put(
      this.BASE_URL + "/tasks/" + playbookId,
      payload
    );
    return res.data;
  };
}
