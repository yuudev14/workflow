"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import AdminService from "@/services/admin/admin";
import { UserWithRoles } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Create and edit share this dialog because the fields are the same. They are
 * not the same request, though: creating takes the password and role ids in one
 * body, while editing sends the profile and the roles separately — the api has
 * no endpoint that changes both at once.
 */
export function UserDialog({
  open,
  onOpenChange,
  user,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: UserWithRoles | null;
}) {
  const queryClient = useQueryClient();
  const editing = user !== null;

  const [username, setUsername] = React.useState("");
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");
  const [roleIds, setRoleIds] = React.useState<string[]>([]);

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: AdminService.listRoles,
    enabled: open,
  });
  const roles = rolesQuery.data ?? [];

  // Reset from the record every time the dialog opens, so a cancelled edit
  // never leaks into the next one.
  React.useEffect(() => {
    if (!open) return;
    setUsername(user?.username ?? "");
    setEmail(user?.email ?? "");
    setFirstName(user?.first_name ?? "");
    setLastName(user?.last_name ?? "");
    setPassword("");
    setRoleIds([]);
  }, [open, user]);

  // Role ids need the roles list to resolve the names the user row carries.
  React.useEffect(() => {
    if (!open || !user || roles.length === 0) return;
    setRoleIds(roles.filter((r) => user.roles.includes(r.name)).map((r) => r.id));
  }, [open, user, roles]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!editing) {
        await AdminService.createUser({
          username,
          email,
          password,
          first_name: firstName || null,
          last_name: lastName || null,
          role_ids: roleIds,
        });
        return;
      }
      await AdminService.updateUser(user.id, {
        email,
        first_name: firstName || null,
        last_name: lastName || null,
      });
      await AdminService.setUserRoles(user.id, roleIds);
    },
    onSuccess: () => {
      toast({ variant: "success", title: editing ? "User updated" : "User created" });
      queryClient.invalidateQueries({ queryKey: ["users"] });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: editing ? "Couldn't update the user" : "Couldn't create the user",
        description: apiErrorMessage(error),
      }),
  });

  const toggleRole = (id: string) =>
    setRoleIds((prev) => (prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? `Edit ${user.username}` : "New user"}</DialogTitle>
          <DialogDescription>
            {editing
              ? "The username cannot change — it identifies the account in the audit trail."
              : "The user signs in with this username and password."}
          </DialogDescription>
        </DialogHeader>

        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              value={username}
              disabled={editing}
              required
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              value={email}
              required
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          {!editing && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                required
                minLength={8}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="first_name">First name</Label>
              <Input
                id="first_name"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="last_name">Last name</Label>
              <Input
                id="last_name"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Roles</Label>
            <div className="flex flex-wrap gap-1.5">
              {roles.map((role) => (
                <button
                  key={role.id}
                  type="button"
                  onClick={() => toggleRole(role.id)}
                  className={
                    roleIds.includes(role.id)
                      ? "rounded-sm border border-signal-dot bg-paper-sunken px-2 py-1 text-[12.5px] font-semibold"
                      : "rounded-sm border border-line-strong px-2 py-1 text-[12.5px] text-ink-soft hover:bg-paper-sunken"
                  }
                >
                  {role.name}
                </button>
              ))}
              {roles.length === 0 && <span className="text-xs text-ink-faint">No roles yet</span>}
            </div>
          </div>

          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending} showLoader={mutation.isPending}>
              {editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
