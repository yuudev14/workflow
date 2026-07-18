import { Alert, AlertsSummary } from "./alerts.schema";
import { ALERTS, ALERTS_SUMMARY } from "./alerts.mock";

/**
 * Alerts data.
 *
 * PHASE 1: returns mock fixtures — there is no alerts backend yet.
 * PHASE 3: replace each body with an `apiClient` call. Signatures and return
 * types stay identical so the pages don't change.
 */
export default class AlertService {
  public static getAlerts = async (): Promise<Alert[]> => Promise.resolve(ALERTS);

  public static getAlertById = async (id: string): Promise<Alert | undefined> =>
    Promise.resolve(ALERTS.find((a) => a.id === id));

  public static getAlertsSummary = async (): Promise<AlertsSummary> =>
    Promise.resolve(ALERTS_SUMMARY);
}
