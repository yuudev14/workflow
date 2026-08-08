"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import AdminService from "@/services/admin/admin";
import { useAuth } from "@/components/provider/auth-provider";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { InitialsAvatar } from "./InitialsAvatar";

const UNASSIGNED = "__unassigned__";

/** Assignee picker, backed by the ungated `GET /users/v1/assignable`. */
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

  const usersQuery = useQuery({
    queryKey: ["assignable-users"],
    queryFn: () => AdminService.listAssignableUsers(),
    enabled: canAssign,
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
        {(usersQuery.data ?? []).map((u) => (
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
