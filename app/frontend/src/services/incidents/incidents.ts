import { Incident, IncidentsSummary } from "./incidents.schema";
import { INCIDENTS, INCIDENTS_SUMMARY } from "./incidents.mock";

/**
 * Incidents data.
 *
 * PHASE 1: returns mock fixtures — there is no incidents backend yet.
 * PHASE 3: replace each body with an `apiClient` call; signatures/return types
 * are unchanged so the pages don't move.
 */
export default class IncidentService {
  public static getIncidents = async (): Promise<Incident[]> =>
    Promise.resolve(INCIDENTS);

  public static getIncidentById = async (id: string): Promise<Incident | undefined> =>
    Promise.resolve(INCIDENTS.find((i) => i.id === id));

  public static getIncidentsSummary = async (): Promise<IncidentsSummary> =>
    Promise.resolve(INCIDENTS_SUMMARY);
}
