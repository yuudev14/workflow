import settings from "@/settings";
import apiClient from "../common/client";
import { CreateSchedulePayload, Schedule, UpdateSchedulePayload } from "./schedules.schema";

export default class ScheduleService {
  private static BASE_URL = settings.BASE_URL.PLAYBOOK_SERVICE_API + "/api/v1/schedules";

  public static list = async (playbookId?: string): Promise<Schedule[]> => {
    const res = await apiClient.get(this.BASE_URL, { params: { playbook_id: playbookId } });
    return res.data;
  };

  public static create = async (payload: CreateSchedulePayload): Promise<Schedule> => {
    const res = await apiClient.post(this.BASE_URL, payload);
    return res.data;
  };

  public static update = async (id: string, payload: UpdateSchedulePayload): Promise<Schedule> => {
    const res = await apiClient.patch(`${this.BASE_URL}/${id}`, payload);
    return res.data;
  };

  public static remove = async (id: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/${id}`);
  };

  /** The api computes the next run times, so the UI needs no cron library. */
  public static preview = async (cron: string, timezone: string): Promise<string[]> => {
    const res = await apiClient.post(`${this.BASE_URL}/preview`, { cron, timezone });
    return res.data.next_runs;
  };
}
