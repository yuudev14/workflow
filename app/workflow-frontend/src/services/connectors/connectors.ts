import settings from "@/settings";
import { EntryResponse } from "../common/schema";
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
      this.BASE_URL + "/api/connectors",
    );
    return res.data;
  };

}