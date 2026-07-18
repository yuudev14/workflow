import { KpiMetric } from "./metrics.schema";
import { ALERT_KPIS, INCIDENT_KPIS, PLAYBOOK_KPIS } from "./metrics.mock";

/**
 * Dashboard KPI aggregates.
 *
 * PHASE 1: returns mock fixtures. There is no aggregate endpoint yet.
 * PHASE 3: replace each body with an `apiClient` call — the method signatures
 * and return types stay the same, so the dashboards don't change.
 */
export default class MetricsService {
  public static getPlaybookKpis = async (): Promise<KpiMetric[]> =>
    Promise.resolve(PLAYBOOK_KPIS);

  public static getAlertKpis = async (): Promise<KpiMetric[]> =>
    Promise.resolve(ALERT_KPIS);

  public static getIncidentKpis = async (): Promise<KpiMetric[]> =>
    Promise.resolve(INCIDENT_KPIS);
}
