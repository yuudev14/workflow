"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, ShieldCheck, Trash2 } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { Role } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import { EmptyState, PageShell } from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Chip } from "../_components/Chip";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { Matrix, PermissionMatrix } from "./_components/PermissionMatrix";

export default function RolesPage() {
  const queryClient = useQueryClient();
  const canCreate = usePermission("settings", "create");
  const canUpdate = usePermission("settings", "update");
  const canDelete = usePermission("settings", "delete");

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [draft, setDraft] = React.useState<Matrix>({});
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<Role | null>(null);

  const rolesQuery = useQuery({ queryKey: ["roles"], queryFn: AdminService.listRoles });
  const roles = rolesQuery.data ?? [];
  const selected = roles.find((r) => r.id === selectedId) ?? roles[0] ?? null;

  // The draft mirrors the selected role until the user edits it; re-seeding on
  // identity keeps a save on one role from carrying into the next.
  React.useEffect(() => {
    setDraft(selected?.permissions ?? {});
  }, [selected?.id, selected?.permissions]);

  // Builtins are the seeded admin/analyst/viewer; the api rejects changes to
  // them, so the editor shows them read-only rather than failing on save.
  const readOnly = !canUpdate || !selected || selected.is_builtin;

  const savePermissions = useMutation({
    mutationFn: () => AdminService.setRolePermissions(selected!.id, draft),
    onSuccess: () => {
      toast({
        variant: "success",
        title: "Permissions updated",
        description: "Applies immediately — no one needs to sign in again.",
      });
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't save the permissions",
        description: apiErrorMessage(error),
      }),
  });

  const remove = useMutation({
    mutationFn: () => AdminService.deleteRole(deleting!.id),
    onSuccess: () => {
      toast({ variant: "success", title: "Role deleted" });
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      setDeleting(null);
      setSelectedId(null);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't delete the role",
        description: apiErrorMessage(error),
      }),
  });

  return (
    <PageShell
      title="Roles"
      subtitle="What each role may do. Changes take effect on the next request."
      actions={
        canCreate && (
          <Button onClick={() => setCreating(true)}>
            <Plus /> New role
          </Button>
        )
      }
    >
      {rolesQuery.isLoading ? (
        <Skeleton className="h-64 rounded-md" />
      ) : roles.length === 0 ? (
        <EmptyState icon={ShieldCheck} title="No roles" description="Create the first role." />
      ) : (
        <div className="flex flex-col gap-3.5 lg:flex-row">
          <div className="w-full shrink-0 overflow-hidden rounded-md border border-line lg:w-[260px]">
            {roles.map((role) => (
              <button
                key={role.id}
                onClick={() => setSelectedId(role.id)}
                className={
                  "flex w-full flex-col gap-0.5 border-b border-line px-3 py-2.5 text-left last:border-b-0 hover:bg-paper-sunken " +
                  (selected?.id === role.id ? "bg-paper-sunken" : "")
                }
              >
                <span className="flex items-center gap-1.5 text-[13.5px] font-semibold">
                  {role.name}
                  {role.is_builtin && <Chip>builtin</Chip>}
                </span>
                <span className="text-[12.5px] text-ink-faint">
                  {role.description ?? "No description"}
                </span>
              </button>
            ))}
          </div>

          {selected && (
            <div className="flex min-w-0 flex-1 flex-col gap-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-[15px] font-semibold">{selected.name}</div>
                  <p className="mt-0.5 text-[12.5px] text-ink-faint">
                    {selected.is_builtin
                      ? "Builtin roles are read-only — clone the grants into a new role to customise."
                      : "Click a module name to toggle its whole row."}
                  </p>
                </div>
                {canDelete && !selected.is_builtin && (
                  <Button variant="ghost" size="icon" title="Delete role" onClick={() => setDeleting(selected)}>
                    <Trash2 />
                  </Button>
                )}
              </div>

              <PermissionMatrix value={draft} onChange={setDraft} readOnly={readOnly} />

              {!readOnly && (
                <div className="flex justify-end gap-2">
                  <Button variant="outline" onClick={() => setDraft(selected.permissions ?? {})}>
                    Reset
                  </Button>
                  <Button
                    disabled={savePermissions.isPending}
                    showLoader={savePermissions.isPending}
                    onClick={() => savePermissions.mutate()}
                  >
                    Save permissions
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <CreateRoleDialog open={creating} onOpenChange={setCreating} />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title={`Delete ${deleting?.name}?`}
        description="Users holding this role lose its grants immediately."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </PageShell>
  );
}

function CreateRoleDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [permissions, setPermissions] = React.useState<Matrix>({});

  React.useEffect(() => {
    if (!open) return;
    setName("");
    setDescription("");
    setPermissions({});
  }, [open]);

  const create = useMutation({
    mutationFn: () =>
      AdminService.createRole({ name, description: description || null, permissions }),
    onSuccess: () => {
      toast({ variant: "success", title: "Role created" });
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't create the role",
        description: apiErrorMessage(error),
      }),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>New role</DialogTitle>
          <DialogDescription>Pick the grants now; they can be changed later.</DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            create.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="role_name">Name</Label>
            <Input
              id="role_name"
              value={name}
              required
              onChange={(e) => setName(e.target.value)}
              placeholder="tier-1-analyst"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="role_description">Description</Label>
            <Textarea
              id="role_description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <PermissionMatrix value={permissions} onChange={setPermissions} />
          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={create.isPending} showLoader={create.isPending}>
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
