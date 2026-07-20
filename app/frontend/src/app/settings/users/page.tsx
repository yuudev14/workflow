"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Pencil, Plus, UserCheck, UserMinus, Users } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { UserWithRoles } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { useAuth } from "@/components/provider/auth-provider";
import { usePermission } from "@/hooks/usePermission";
import { EmptyState, PageShell, SearchInput } from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Chip } from "../_components/Chip";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { PasswordDialog } from "./_components/PasswordDialog";
import { UserDialog } from "./_components/UserDialog";

export default function UsersPage() {
  const queryClient = useQueryClient();
  const { user: currentUser } = useAuth();
  const canCreate = usePermission("settings", "create");
  const canUpdate = usePermission("settings", "update");
  const canDelete = usePermission("settings", "delete");

  const [search, setSearch] = React.useState("");
  const [editing, setEditing] = React.useState<UserWithRoles | null>(null);
  const [dialog, setDialog] = React.useState<"none" | "form" | "password" | "deactivate">("none");

  const usersQuery = useQuery({
    queryKey: ["users", search],
    queryFn: () => AdminService.listUsers({ search: search || undefined, limit: 100 }),
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
      <SearchInput
        placeholder="Search username or email…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

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
          description={search ? "Try a different search." : "Create the first account."}
        />
      ) : (
        <div className="overflow-x-auto rounded-md border border-line">
          <table className="w-full text-[13.5px]">
            <thead className="bg-paper-sunken text-[12px] uppercase tracking-wide text-ink-soft">
              <tr>
                <th className="px-3 py-2 text-left font-semibold">User</th>
                <th className="px-3 py-2 text-left font-semibold">Roles</th>
                <th className="px-3 py-2 text-left font-semibold">Provider</th>
                <th className="px-3 py-2 text-left font-semibold">Status</th>
                <th className="px-3 py-2 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id} className="border-t border-line">
                  <td className="px-3 py-2.5">
                    <div className="font-semibold">{user.username}</div>
                    <div className="text-[12.5px] text-ink-faint">{user.email}</div>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex flex-wrap gap-1">
                      {user.roles.length === 0 ? (
                        <span className="text-ink-faint">—</span>
                      ) : (
                        user.roles.map((role) => <Chip key={role}>{role}</Chip>)
                      )}
                    </div>
                  </td>
                  <td className="px-3 py-2.5 text-ink-soft">{user.auth_provider}</td>
                  <td className="px-3 py-2.5">
                    <Chip
                      className={
                        user.is_active
                          ? "border-signal-dot text-foreground"
                          : "border-line text-ink-faint"
                      }
                    >
                      {user.is_active ? "active" : "inactive"}
                    </Chip>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex justify-end gap-1">
                      {canUpdate && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            title="Edit"
                            onClick={() => open(user, "form")}
                          >
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
                      {/* Deactivating yourself is rejected by the api too — this just
                          keeps the button from being offered. */}
                      {canDelete && user.is_active && user.id !== currentUser?.id && (
                        <Button
                          variant="ghost"
                          size="icon"
                          title="Deactivate"
                          onClick={() => open(user, "deactivate")}
                        >
                          <UserMinus />
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
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
        description="They are signed out everywhere and cannot sign in again until reactivated. The account is kept — the audit trail references it."
        confirmLabel="Deactivate"
        pending={deactivate.isPending}
        onConfirm={() => deactivate.mutate()}
      />
    </PageShell>
  );
}
