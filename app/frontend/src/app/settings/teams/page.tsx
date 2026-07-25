"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { Pencil, Plus, Trash2, UsersRound, X } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { Team, TeamMember } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import {
  DataTable,
  EmptyState,
  Glyph,
  InitialsAvatar,
  PageShell,
  SearchInput,
} from "@/components/soar";
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
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { Chip } from "../_components/Chip";

const columnHelper = createColumnHelper<Team>();

export default function TeamsPage() {
  const queryClient = useQueryClient();
  const canCreate = usePermission("settings", "create");
  const canUpdate = usePermission("settings", "update");
  const canDelete = usePermission("settings", "delete");

  const [search, setSearch] = React.useState("");
  const [editing, setEditing] = React.useState<Team | null>(null);
  const [formOpen, setFormOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState<Team | null>(null);

  const teamsQuery = useQuery({
    queryKey: ["teams", search],
    queryFn: () => AdminService.listTeams({ search: search || undefined, limit: 100 }),
  });
  const teams = teamsQuery.data?.entries ?? [];

  const remove = useMutation({
    mutationFn: () => AdminService.deleteTeam(deleting!.id),
    onSuccess: () => {
      toast({ variant: "success", title: "Team deleted" });
      queryClient.invalidateQueries({ queryKey: ["teams"] });
      setDeleting(null);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't delete the team",
        description: apiErrorMessage(error),
      }),
  });

  const open = (team: Team | null) => {
    setEditing(team);
    setFormOpen(true);
  };

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Team",
        cell: ({ row: { original: team } }) => (
          <div className="flex min-w-0 items-start gap-2.5">
            <Glyph icon={UsersRound} tone="slate" size="md" className="mt-0.5" />
            <div className="min-w-0">
              <div className="font-semibold">{team.name}</div>
              <div className="truncate text-[12.5px] text-ink-faint">
                {team.description ?? "No description"}
              </div>
            </div>
          </div>
        ),
      }),
      columnHelper.accessor("members", {
        header: "Members",
        cell: ({ getValue }) => {
          const members = getValue();
          if (members.length === 0) {
            return <span className="text-[12.5px] text-ink-faint">No members</span>;
          }
          return (
            <div className="flex items-center">
              <div className="flex">
                {members.slice(0, 6).map((m) => (
                  <InitialsAvatar
                    key={m.id}
                    name={m.username}
                    size={24}
                    className="-ml-2 ring-2 ring-card first:ml-0"
                  />
                ))}
              </div>
              <span className="ml-2 text-[12.5px] text-ink-faint tnum">
                {members.length > 6 ? `+${members.length - 6} more` : members.length}
              </span>
            </div>
          );
        },
      }),
      // Grants are visible at a glance because team membership escalates
      // privilege — every member inherits these roles.
      columnHelper.accessor("roles", {
        header: "Grants",
        cell: ({ getValue }) => {
          const roles = getValue() ?? [];
          if (roles.length === 0) {
            return <span className="text-[12.5px] text-ink-faint">No grants</span>;
          }
          return (
            <div className="flex flex-wrap gap-1">
              {roles.map((r) => (
                <Chip key={r.id}>{r.name}</Chip>
              ))}
            </div>
          );
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        meta: { align: "right" },
        cell: ({ row: { original: team } }) => (
          <div className="flex justify-end gap-1">
            {canUpdate && (
              <Button variant="ghost" size="icon" title="Edit" onClick={() => open(team)}>
                <Pencil />
              </Button>
            )}
            {canDelete && (
              <Button variant="ghost" size="icon" title="Delete" onClick={() => setDeleting(team)}>
                <Trash2 />
              </Button>
            )}
          </div>
        ),
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canUpdate, canDelete]
  );

  return (
    <PageShell
      title="Teams"
      subtitle="Labels for grouping people. They do not grant anything on their own."
      actions={
        canCreate && (
          <Button onClick={() => open(null)}>
            <Plus /> New team
          </Button>
        )
      }
    >
      <SearchInput
        placeholder="Search teams…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      {teamsQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-md" />
          ))}
        </div>
      ) : teams.length === 0 ? (
        <EmptyState
          icon={UsersRound}
          title="No teams"
          description={search ? "Try a different search." : "Create the first team."}
        />
      ) : (
        <DataTable columns={columns} data={teams} getRowId={(team) => team.id} pageSize={10} />
      )}

      <TeamDialog open={formOpen} onOpenChange={setFormOpen} team={editing} />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title={`Delete ${deleting?.name}?`}
        description="The members keep their accounts and roles; only the grouping goes away."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </PageShell>
  );
}

function TeamDialog({
  open,
  onOpenChange,
  team,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  team: Team | null;
}) {
  const queryClient = useQueryClient();
  const editing = team !== null;

  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [selected, setSelected] = React.useState<TeamMember[]>([]);
  const [memberSearch, setMemberSearch] = React.useState("");
  const [roleIds, setRoleIds] = React.useState<string[]>([]);

  const rolesQuery = useQuery({
    queryKey: ["roles", "team-picker"],
    queryFn: () => AdminService.listRoles(),
    enabled: open,
  });
  const allRoles = rolesQuery.data ?? [];

  // Server-side search keeps the picker usable with thousands of users — we
  // never pull the whole directory into the dialog.
  const usersQuery = useQuery({
    queryKey: ["users", "team-picker", memberSearch],
    queryFn: () => AdminService.listUsers({ search: memberSearch || undefined, limit: 20 }),
    enabled: open,
  });
  const results = usersQuery.data?.entries ?? [];
  const truncated = (usersQuery.data?.total ?? 0) > results.length;

  React.useEffect(() => {
    if (!open) return;
    setName(team?.name ?? "");
    setDescription(team?.description ?? "");
    setSelected(team?.members ?? []);
    setMemberSearch("");
    setRoleIds((team?.roles ?? []).map((r) => r.id));
  }, [open, team]);

  const save = useMutation({
    mutationFn: async () => {
      const memberIds = selected.map((m) => m.id);
      if (!editing) {
        await AdminService.createTeam({
          name,
          description: description || null,
          member_ids: memberIds,
          role_ids: roleIds,
        });
        return;
      }
      await AdminService.updateTeam(team.id, { name, description: description || null });
      await AdminService.setTeamMembers(team.id, memberIds);
      await AdminService.setTeamRoles(team.id, roleIds);
    },
    onSuccess: () => {
      toast({ variant: "success", title: editing ? "Team updated" : "Team created" });
      queryClient.invalidateQueries({ queryKey: ["teams"] });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't save the team",
        description: apiErrorMessage(error),
      }),
  });

  const toggle = (member: TeamMember) =>
    setSelected((prev) =>
      prev.some((m) => m.id === member.id)
        ? prev.filter((m) => m.id !== member.id)
        : [...prev, member]
    );

  const toggleRole = (id: string) =>
    setRoleIds((prev) => (prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? `Edit ${team.name}` : "New team"}</DialogTitle>
          <DialogDescription>Membership is replaced with exactly what is selected.</DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            save.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="team_name">Name</Label>
            <Input id="team_name" value={name} required onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="team_description">Description</Label>
            <Textarea
              id="team_description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <div className="flex items-baseline justify-between">
              <Label>Roles granted to members</Label>
              <span className="text-[12px] text-ink-faint tnum">{roleIds.length} selected</span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {allRoles.map((role) => {
                const on = roleIds.includes(role.id);
                return (
                  <button
                    key={role.id}
                    type="button"
                    onClick={() => toggleRole(role.id)}
                    aria-pressed={on}
                    className={
                      "rounded-full border px-2.5 py-1 text-[12.5px] font-medium transition-colors " +
                      (on
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-line bg-paper-sunken text-ink-faint hover:border-line-strong")
                    }
                  >
                    {role.name}
                  </button>
                );
              })}
            </div>
            <span className="text-[12px] text-ink-faint">
              Everyone in this team inherits these roles, on top of their own.
            </span>
          </div>

          <div className="flex flex-col gap-1.5">
            <div className="flex items-baseline justify-between">
              <Label>Members</Label>
              <span className="text-[12px] text-ink-faint tnum">
                {selected.length} selected
              </span>
            </div>

            {selected.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {selected.map((member) => (
                  <button
                    key={member.id}
                    type="button"
                    onClick={() => toggle(member)}
                    className="flex items-center gap-1.5 rounded-full border border-line bg-paper-sunken py-0.5 pr-1.5 pl-1 text-[12.5px] hover:border-line-strong"
                    title="Remove"
                  >
                    <InitialsAvatar name={member.username} size={18} />
                    <span className="font-medium">{member.username}</span>
                    <X className="size-3.5 text-ink-faint" />
                  </button>
                ))}
              </div>
            )}

            <SearchInput
              placeholder="Search users to add…"
              value={memberSearch}
              onChange={(e) => setMemberSearch(e.target.value)}
            />
            <div className="flex max-h-52 flex-col gap-0.5 overflow-y-auto rounded-md border border-line p-1.5">
              {results.map((user) => {
                const checked = selected.some((m) => m.id === user.id);
                return (
                  <label
                    key={user.id}
                    className="flex items-center gap-2.5 rounded-sm px-1.5 py-1.5 text-[13px] hover:bg-paper-sunken"
                  >
                    <input
                      type="checkbox"
                      className="size-4 accent-primary"
                      checked={checked}
                      onChange={() =>
                        toggle({ id: user.id, username: user.username, email: user.email })
                      }
                    />
                    <InitialsAvatar name={user.username} size={22} />
                    <span className="font-medium">{user.username}</span>
                    <span className="text-ink-faint">{user.email}</span>
                  </label>
                );
              })}
              {results.length === 0 && (
                <span className="px-1.5 py-2 text-xs text-ink-faint">
                  {usersQuery.isLoading ? "Searching…" : "No users match."}
                </span>
              )}
              {truncated && (
                <span className="px-1.5 py-1.5 text-[12px] text-ink-faint">
                  More users exist — refine your search to find them.
                </span>
              )}
            </div>
          </div>
          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending} showLoader={save.isPending}>
              {editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
