"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2, UsersRound } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { Team } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import { EmptyState, PageShell, SearchInput } from "@/components/soar";
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
        <div className="overflow-hidden rounded-md border border-line">
          {teams.map((team) => (
            <div
              key={team.id}
              className="flex items-start justify-between gap-3 border-b border-line px-3.5 py-3 last:border-b-0"
            >
              <div className="min-w-0">
                <div className="text-[14px] font-semibold">{team.name}</div>
                <div className="text-[12.5px] text-ink-faint">
                  {team.description ?? "No description"}
                </div>
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {team.members.length === 0 ? (
                    <span className="text-[12.5px] text-ink-faint">No members</span>
                  ) : (
                    team.members.map((m) => <Chip key={m.id}>{m.username}</Chip>)
                  )}
                </div>
              </div>
              <div className="flex shrink-0 gap-1">
                {canUpdate && (
                  <Button variant="ghost" size="icon" title="Edit" onClick={() => open(team)}>
                    <Pencil />
                  </Button>
                )}
                {canDelete && (
                  <Button
                    variant="ghost"
                    size="icon"
                    title="Delete"
                    onClick={() => setDeleting(team)}
                  >
                    <Trash2 />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
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
  const [memberIds, setMemberIds] = React.useState<string[]>([]);

  const usersQuery = useQuery({
    queryKey: ["users", ""],
    queryFn: () => AdminService.listUsers({ limit: 100 }),
    enabled: open,
  });
  const users = usersQuery.data?.entries ?? [];

  React.useEffect(() => {
    if (!open) return;
    setName(team?.name ?? "");
    setDescription(team?.description ?? "");
    setMemberIds(team?.members.map((m) => m.id) ?? []);
  }, [open, team]);

  const save = useMutation({
    mutationFn: async () => {
      if (!editing) {
        await AdminService.createTeam({
          name,
          description: description || null,
          member_ids: memberIds,
        });
        return;
      }
      await AdminService.updateTeam(team.id, { name, description: description || null });
      await AdminService.setTeamMembers(team.id, memberIds);
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

  const toggle = (id: string) =>
    setMemberIds((prev) => (prev.includes(id) ? prev.filter((m) => m !== id) : [...prev, id]));

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
            <Label>Members</Label>
            <div className="flex max-h-52 flex-col gap-1 overflow-y-auto rounded-md border border-line p-2">
              {users.map((user) => (
                <label key={user.id} className="flex items-center gap-2 text-[13px]">
                  <input
                    type="checkbox"
                    className="size-4 accent-primary"
                    checked={memberIds.includes(user.id)}
                    onChange={() => toggle(user.id)}
                  />
                  <span className="font-medium">{user.username}</span>
                  <span className="text-ink-faint">{user.email}</span>
                </label>
              ))}
              {users.length === 0 && <span className="text-xs text-ink-faint">No users</span>}
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
