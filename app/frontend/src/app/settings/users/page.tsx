"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { KeyRound, Pencil, Plus, UserCheck, UserMinus, Users } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { UserWithRoles } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { useAuth } from "@/components/provider/auth-provider";
import { usePermission } from "@/hooks/usePermission";
import {
  DataTable,
  EmptyState,
  FilterChips,
  InitialsAvatar,
  PageShell,
  SearchInput,
  StatusPill,
} from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Chip } from "../_components/Chip";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { PasswordDialog } from "./_components/PasswordDialog";
import { UserDialog } from "./_components/UserDialog";

const columnHelper = createColumnHelper<UserWithRoles>();

export default function UsersPage() {
  const queryClient = useQueryClient();
  const { user: currentUser } = useAuth();
  const canCreate = usePermission("settings", "create");
  const canUpdate = usePermission("settings", "update");
  const canDelete = usePermission("settings", "delete");

  const [search, setSearch] = React.useState("");
  const [status, setStatus] = React.useState<"" | "active" | "inactive">("");
  const [role, setRole] = React.useState("all");
  const [editing, setEditing] = React.useState<UserWithRoles | null>(null);
  const [dialog, setDialog] = React.useState<"none" | "form" | "password" | "deactivate">("none");

  const rolesQuery = useQuery({ queryKey: ["roles"], queryFn: AdminService.listRoles });

  const usersQuery = useQuery({
    queryKey: ["users", search, status, role],
    queryFn: () =>
      AdminService.listUsers({
        search: search || undefined,
        is_active: status === "" ? undefined : status === "active",
        role: role === "all" ? undefined : role,
        limit: 100,
      }),
  });
  const users = usersQuery.data?.entries ?? [];

  const deactivate = useMutation({
    mutationFn: () => AdminService.deactivateUser(editing!.id),
    onSuccess: () => {
      toast({ variant: "success", title: "User deactivated" });
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setDialog("none");
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't deactivate the user",
        description: apiErrorMessage(error),
      }),
  });

  const reactivate = useMutation({
    mutationFn: (user: UserWithRoles) => AdminService.updateUser(user.id, { is_active: true }),
    onSuccess: () => {
      toast({ variant: "success", title: "User reactivated" });
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't reactivate the user",
        description: apiErrorMessage(error),
      }),
  });

  const open = (user: UserWithRoles | null, which: "form" | "password" | "deactivate") => {
    setEditing(user);
    setDialog(which);
  };

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("username", {
        header: "User",
        cell: ({ row: { original: user } }) => (
          <div className="flex items-center gap-2.5">
            <InitialsAvatar name={user.username} size={28} />
            <div className="min-w-0">
              <div className="font-semibold">{user.username}</div>
              <div className="truncate text-[12.5px] text-ink-faint">{user.email}</div>
            </div>
          </div>
        ),
      }),
      columnHelper.accessor("roles", {
        header: "Roles",
        cell: ({ getValue }) => {
          const roles = getValue();
          return (
            <div className="flex flex-wrap gap-1">
              {roles.length === 0 ? (
                <span className="text-ink-faint">-</span>
              ) : (
                roles.map((role) => <Chip key={role}>{role}</Chip>)
              )}
            </div>
          );
        },
      }),
      columnHelper.accessor("auth_provider", {
        header: "Provider",
        cell: ({ getValue }) => <span className="text-ink-soft">{getValue()}</span>,
      }),
      columnHelper.accessor("is_active", {
        header: "Status",
        cell: ({ getValue }) => (
          <StatusPill variant={getValue() ? "success" : "neutral"}>
            {getValue() ? "Active" : "Inactive"}
          </StatusPill>
        ),
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        meta: { align: "right" },
        cell: ({ row: { original: user } }) => (
          <div className="flex justify-end gap-1">
            {canUpdate && (
              <>
                <Button variant="ghost" size="icon" title="Edit" onClick={() => open(user, "form")}>
                  <Pencil />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  title="Reset password"
                  onClick={() => open(user, "password")}
                >
                  <KeyRound />
                </Button>
              </>
            )}
            {canUpdate && !user.is_active && (
              <Button
                variant="ghost"
                size="icon"
                title="Reactivate"
                disabled={reactivate.isPending}
                onClick={() => reactivate.mutate(user)}
              >
                <UserCheck />
              </Button>
            )}
            {/* Deactivating yourself is rejected by the api too - this just
                keeps the button from being offered. */}
            {canDelete && user.is_active && user.id !== currentUser?.id && (
              <Button variant="ghost" size="icon" title="Deactivate" onClick={() => open(user, "deactivate")}>
                <UserMinus />
              </Button>
            )}
          </div>
        ),
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canUpdate, canDelete, currentUser?.id, reactivate.isPending]
  );

  return (
    <PageShell
      title="Users"
      subtitle="Accounts, their roles, and whether they can sign in."
      actions={
        canCreate && (
          <Button onClick={() => open(null, "form")}>
            <Plus /> New user
          </Button>
        )
      }
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SearchInput
          placeholder="Search username or email…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-sm"
        />
        <div className="flex flex-wrap items-center gap-3">
          <Select value={role} onValueChange={setRole}>
            <SelectTrigger className="w-44" aria-label="Filter by role">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All roles</SelectItem>
              {rolesQuery.data?.map((r) => (
                <SelectItem key={r.id} value={r.name}>
                  {r.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FilterChips
            value={status}
            onChange={(v) => setStatus(v as typeof status)}
            chips={[
              { value: "", label: "All" },
              { value: "active", label: "Active" },
              { value: "inactive", label: "Inactive" },
            ]}
          />
        </div>
      </div>

      {usersQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-md" />
          ))}
        </div>
      ) : users.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No users found"
          description={
            search || status || role !== "all"
              ? "No users match these filters."
              : "Create the first account."
          }
        />
      ) : (
        <DataTable columns={columns} data={users} getRowId={(user) => user.id} pageSize={10} />
      )}

      <UserDialog
        open={dialog === "form"}
        onOpenChange={(next) => setDialog(next ? "form" : "none")}
        user={editing}
      />
      <PasswordDialog
        open={dialog === "password"}
        onOpenChange={(next) => setDialog(next ? "password" : "none")}
        user={editing}
      />
      <ConfirmDialog
        open={dialog === "deactivate"}
        onOpenChange={(next) => setDialog(next ? "deactivate" : "none")}
        title={`Deactivate ${editing?.username}?`}
        description="They are signed out everywhere and cannot sign in again until reactivated. The account is kept - the audit trail references it."
        confirmLabel="Deactivate"
        pending={deactivate.isPending}
        onConfirm={() => deactivate.mutate()}
      />
    </PageShell>
  );
}
