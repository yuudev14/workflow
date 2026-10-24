import settings from "@/settings";
import {
  ConnectorInfo,
} from "./connectors.schema";
import apiClient from "../common/client";

export default class ConnectorService {
  private static BASE_URL =
    settings.BASE_URL.CONNECTORS_SERVICE_API;

  /**
   * get available connectors lists
   */
  public static getConnectors = async (
  ): Promise<ConnectorInfo[]> => {
    const res = await apiClient.get(
      this.BASE_URL + "/api/v1/connectors",
    );
    return res.data;
  };

  /**
   * upload a connector zip. Without replace, the api returns 409 when the
   * connector is already installed.
   */
  public static uploadConnector = async (
    file: File,
    replace = false,
  ): Promise<ConnectorInfo> => {
    const form = new FormData();
    form.append("file", file);
    const res = await apiClient.post(
      this.BASE_URL + "/api/v1/connectors",
      form,
      { params: replace ? { replace: true } : undefined },
    );
    return res.data;
  };
}