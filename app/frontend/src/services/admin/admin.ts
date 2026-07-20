import settings from "@/settings";
import apiClient from "../common/client";
import { EntryResponse } from "../common/schema";
import {
  AuditFilter,
  AuditLog,
  CreateUserPayload,
  Role,
  RolePayload,
  Team,
  TeamFilter,
  TeamPayload,
  UpdateRolePayload,
  UpdateTeamPayload,
  UpdateUserPayload,
  UserFilter,
  UserWithRoles,
} from "./admin.schema";

/**
 * The settings module: users, roles, teams and the audit trail. They share the
 * `settings.*` grants and the same api binary as auth, hence the same base url.
 */
export default class AdminService {
  private static BASE_URL = settings.BASE_URL.AUTH_SERVICE_API + "/api";

  // ---- users ----

  public static listUsers = async (filter: UserFilter = {}): Promise<EntryResponse<UserWithRoles>> => {
    const res = await apiClient.get(`${this.BASE_URL}/users/v1`, { params: filter });
    return res.data;
  };

  public static getUser = async (id: string): Promise<UserWithRoles> => {
    const res = await apiClient.get(`${this.BASE_URL}/users/v1/${id}`);
    return res.data;
  };

  public static createUser = async (payload: CreateUserPayload): Promise<UserWithRoles> => {
    const res = await apiClient.post(`${this.BASE_URL}/users/v1`, payload);
    return res.data;
  };

  public static updateUser = async (
    id: string,
    payload: UpdateUserPayload,
  ): Promise<UserWithRoles> => {
    const res = await apiClient.put(`${this.BASE_URL}/users/v1/${id}`, payload);
    return res.data;
  };

  public static setUserRoles = async (id: string, roleIds: string[]): Promise<UserWithRoles> => {
    const res = await apiClient.put(`${this.BASE_URL}/users/v1/${id}/roles`, { role_ids: roleIds });
    return res.data;
  };

  public static setUserPassword = async (id: string, password: string): Promise<void> => {
    await apiClient.put(`${this.BASE_URL}/users/v1/${id}/password`, { password });
  };

  /** DELETE deactivates: users are never hard-deleted, the audit trail references them. */
  public static deactivateUser = async (id: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/users/v1/${id}`);
  };

  // ---- roles ----

  public static listRoles = async (): Promise<Role[]> => {
    const res = await apiClient.get(`${this.BASE_URL}/roles/v1`);
    return res.data;
  };

  public static getRole = async (id: string): Promise<Role> => {
    const res = await apiClient.get(`${this.BASE_URL}/roles/v1/${id}`);
    return res.data;
  };

  public static createRole = async (payload: RolePayload): Promise<Role> => {
    const res = await apiClient.post(`${this.BASE_URL}/roles/v1`, payload);
    return res.data;
  };

  public static updateRole = async (id: string, payload: UpdateRolePayload): Promise<Role> => {
    const res = await apiClient.put(`${this.BASE_URL}/roles/v1/${id}`, payload);
    return res.data;
  };

  public static setRolePermissions = async (
    id: string,
    permissions: Record<string, string[]>,
  ): Promise<Role> => {
    const res = await apiClient.put(`${this.BASE_URL}/roles/v1/${id}/permissions`, { permissions });
    return res.data;
  };

  public static deleteRole = async (id: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/roles/v1/${id}`);
  };

  // ---- teams ----

  public static listTeams = async (filter: TeamFilter = {}): Promise<EntryResponse<Team>> => {
    const res = await apiClient.get(`${this.BASE_URL}/teams/v1`, { params: filter });
    return res.data;
  };

  public static getTeam = async (id: string): Promise<Team> => {
    const res = await apiClient.get(`${this.BASE_URL}/teams/v1/${id}`);
    return res.data;
  };

  public static createTeam = async (payload: TeamPayload): Promise<Team> => {
    const res = await apiClient.post(`${this.BASE_URL}/teams/v1`, payload);
    return res.data;
  };

  public static updateTeam = async (id: string, payload: UpdateTeamPayload): Promise<Team> => {
    const res = await apiClient.put(`${this.BASE_URL}/teams/v1/${id}`, payload);
    return res.data;
  };

  public static setTeamMembers = async (id: string, memberIds: string[]): Promise<Team> => {
    const res = await apiClient.put(`${this.BASE_URL}/teams/v1/${id}/members`, {
      member_ids: memberIds,
    });
    return res.data;
  };

  public static deleteTeam = async (id: string): Promise<void> => {
    await apiClient.delete(`${this.BASE_URL}/teams/v1/${id}`);
  };

  // ---- audit ----

  public static listAuditLogs = async (
    filter: AuditFilter = {},
  ): Promise<EntryResponse<AuditLog>> => {
    const res = await apiClient.get(`${this.BASE_URL}/audit/v1`, { params: filter });
    return res.data;
  };
}
