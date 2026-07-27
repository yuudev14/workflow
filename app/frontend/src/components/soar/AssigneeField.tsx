"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import AdminService from "@/services/admin/admin";
import { useAuth } from "@/components/provider/auth-provider";
import { usePermission } from "@/hooks/usePermission";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { InitialsAvatar } from "./InitialsAvatar";

const UNASSIGNED = "__unassigned__";

/**
 * Assignee picker.
 *
 * The full user list comes from `GET /api/users/v1`, which is gated on
 * `settings:read` - an analyst holding only `alerts:update` cannot read it. So
 * the dropdown renders only for users who can list users, and everyone else
 * gets Assign to me / Unassign, which needs no directory at all. Remove the
 * split once a non-settings "assignable users" endpoint exists.
 */
export function AssigneeField({
  assigneeId,
  assignee,
  canAssign,
  pending,
  onAssign,
}: {
  assigneeId?: string | null;
  assignee?: string | null;
  canAssign: boolean;
  pending?: boolean;
  onAssign: (assigneeId: string | null) => void;
}) {
  const { user } = useAuth();
  const canListUsers = usePermission("settings", "read");

  const usersQuery = useQuery({
    queryKey: ["assignable-users"],
    queryFn: () => AdminService.listUsers({ limit: 200 }),
    enabled: canAssign && canListUsers,
    staleTime: 5 * 60 * 1000,
  });

  if (!canAssign) {
    return (
      <div className="flex items-center gap-2 rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13.5px]">
        <InitialsAvatar name={assignee} size={20} />
        {assignee ?? "Unassigned"}
      </div>
    );
  }

  if (canListUsers) {
    return (
      <Select
        value={assigneeId ?? UNASSIGNED}
        disabled={pending}
        onValueChange={(v) => onAssign(v === UNASSIGNED ? null : v)}
      >
        <SelectTrigger className="text-[13.5px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={UNASSIGNED} className="text-[13px]">
            Unassigned
          </SelectItem>
          {(usersQuery.data?.entries ?? []).map((u) => (
            <SelectItem key={u.id} value={u.id} className="text-[13px]">
              <span className="flex items-center gap-2">
                <InitialsAvatar name={u.username} size={18} />
                {u.username}
                {u.id === user?.id && <span className="text-ink-faint">(you)</span>}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  const mine = !!user && assigneeId === user.id;
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-2 rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13.5px]">
        <InitialsAvatar name={assignee} size={20} />
        {assignee ?? "Unassigned"}
      </div>
      <div className="flex gap-2">
        {!mine && user && (
          <button
            onClick={() => onAssign(user.id)}
            disabled={pending}
            className="rounded-sm border border-line-strong px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-50"
          >
            Assign to me
          </button>
        )}
        {assigneeId && (
          <button
            onClick={() => onAssign(null)}
            disabled={pending}
            className="rounded-sm border border-line-strong px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-50"
          >
            Unassign
          </button>
        )}
      </div>
    </div>
  );
}
