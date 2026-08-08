import settings from "@/settings";
import apiClient from "../common/client";
import { CursorPage } from "../common/schema";
import { DateRangeParams } from "../common/range";
import { RunPlaybookPayload } from "../playbooks/playbooks.schema";
import {
  Alert,
  AlertFilter,
  AlertNote,
  AlertsSummary,
  CreateAlertPayload,
  UpdateAlertPayload,
  UpdateAlertStatusPayload,
} from "./alerts.schema";

/**
 * Alerts. Same api binary as auth and playbooks, hence the same base url.
 */
export default class AlertService {
  private static BASE_URL = settings.BASE_URL.AUTH_SERVICE_API + "/api/alerts/v1";

  public static getAlerts = async (filter: AlertFilter = {}): Promise<CursorPage<Alert>> => {
    const res = await apiClient.get(this.BASE_URL, { params: filter });
    return res.data;
  };

  public static getAlertById = async (id: string): Promise<Alert> => {
    const res = await apiClient.get(`${this.BASE_URL}/${id}`);
    return res.data;
  };

  public static getAlertsSummary = async (range: DateRangeParams = {}): Promise<AlertsSummary> => {
    const res = await apiClient.get(`${this.BASE_URL}/summary`, { params: range });
    return res.data;
  };

  /**
   * Runs a playbook against the given alerts - one run for N records, not N
   * runs. Returns 202; the run itself lands on the queue.
   */
  public static runPlaybook = async (payload: RunPlaybookPayload): Promise<unknown> => {
    const res = await apiClient.post(`${this.BASE_URL}/run`, payload);
    return res.data;
  };

  public static createAlert = async (payload: CreateAlertPayload): Promise<Alert> => {
    const res = await apiClient.post(this.BASE_URL, payload);
    return res.data;
  };

  public static updateAlert = async (id: string, payload: UpdateAlertPayload): Promise<Alert> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}`, payload);
    return res.data;
  };

  public static setAlertStatus = async (
    id: string,
    payload: UpdateAlertStatusPayload,
  ): Promise<Alert> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}/status`, payload);
    return res.data;
  };

  /** Creates an incident from the alert and links the two. Returns the incident. */
  public static escalateAlert = async (id: string, title?: string) => {
    const res = await apiClient.post(`${this.BASE_URL}/${id}/escalate`, title ? { title } : {});
    return res.data;
  };

  public static getAlertNotes = async (id: string): Promise<AlertNote[]> => {
    const res = await apiClient.get(`${this.BASE_URL}/${id}/notes`);
    return res.data;
  };

  public static addAlertNote = async (id: string, body: string): Promise<AlertNote> => {
    const res = await apiClient.post(`${this.BASE_URL}/${id}/notes`, { body });
    return res.data;
  };

  public static updateAlertNote = async (
    id: string,
    noteId: string,
    body: string,
  ): Promise<AlertNote> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}/notes/${noteId}`, { body });
    return res.data;
  };

  public static deleteAlertNote = async (id: string, noteId: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/${id}/notes/${noteId}`);
  };
}
