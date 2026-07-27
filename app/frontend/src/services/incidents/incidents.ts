import settings from "@/settings";
import apiClient from "../common/client";
import { CursorPage } from "../common/schema";
import {
  CreateIncidentPayload,
  Incident,
  IncidentFilter,
  IncidentNote,
  IncidentStatus,
  IncidentsSummary,
  UpdateIncidentPayload,
} from "./incidents.schema";

/**
 * Incidents. Same api binary as alerts, hence the same base url.
 */
export default class IncidentService {
  private static BASE_URL = settings.BASE_URL.AUTH_SERVICE_API + "/api/incidents/v1";

  public static getIncidents = async (
    filter: IncidentFilter = {},
  ): Promise<CursorPage<Incident>> => {
    const res = await apiClient.get(this.BASE_URL, { params: filter });
    return res.data;
  };

  public static getIncidentById = async (id: string): Promise<Incident> => {
    const res = await apiClient.get(`${this.BASE_URL}/${id}`);
    return res.data;
  };

  public static getIncidentsSummary = async (): Promise<IncidentsSummary> => {
    const res = await apiClient.get(`${this.BASE_URL}/summary`);
    return res.data;
  };

  public static createIncident = async (payload: CreateIncidentPayload): Promise<Incident> => {
    const res = await apiClient.post(this.BASE_URL, payload);
    return res.data;
  };

  public static updateIncident = async (
    id: string,
    payload: UpdateIncidentPayload,
  ): Promise<Incident> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}`, payload);
    return res.data;
  };

  public static setIncidentStatus = async (
    id: string,
    status: IncidentStatus,
  ): Promise<Incident> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}/status`, { status });
    return res.data;
  };

  public static getIncidentNotes = async (id: string): Promise<IncidentNote[]> => {
    const res = await apiClient.get(`${this.BASE_URL}/${id}/notes`);
    return res.data;
  };

  public static addNote = async (id: string, body: string): Promise<IncidentNote> => {
    const res = await apiClient.post(`${this.BASE_URL}/${id}/notes`, { body });
    return res.data;
  };

  public static updateNote = async (
    id: string,
    noteId: string,
    body: string,
  ): Promise<IncidentNote> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}/notes/${noteId}`, { body });
    return res.data;
  };

  public static deleteNote = async (id: string, noteId: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/${id}/notes/${noteId}`);
  };

  public static linkAlert = async (id: string, alertId: string): Promise<void> => {
    await apiClient.post(`${this.BASE_URL}/${id}/alerts`, { alert_id: alertId });
  };

  public static unlinkAlert = async (id: string, alertId: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/${id}/alerts/${alertId}`);
  };
}
