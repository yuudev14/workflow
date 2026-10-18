import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { Copy, KeySquare, Plus, Trash2 } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { APIKey } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import { DataTable, EmptyState, PageShell } from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfirmDialog } from "./components/ConfirmDialog";

const columnHelper = createColumnHelper<APIKey>();
const date = (value: string | null) => (value ? new Date(value).toLocaleString() : "-");

export default function APIKeysPage() {
  const queryClient = useQueryClient();
  const canCreate = usePermission("settings", "create");
  const canDelete = usePermission("settings", "delete");
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<APIKey | null>(null);

  const keysQuery = useQuery({ queryKey: ["api-keys"], queryFn: AdminService.listAPIKeys });
  const keys = keysQuery.data ?? [];

  const remove = useMutation({
    mutationFn: () => AdminService.deleteAPIKey(deleting!.id),
    onSuccess: () => {
      toast({ variant: "success", title: "API key deleted" });
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setDeleting(null);
    },
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't delete the API key", description: apiErrorMessage(error) }),
  });

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Name",
        cell: ({ row: { original: key } }) => (
          <div>
            <div className="font-semibold">{key.name}</div>
            <div className="font-mono text-[12.5px] text-ink-faint">{key.prefix}…</div>
          </div>
        ),
      }),
      columnHelper.accessor("username", { header: "Acts as" }),
      columnHelper.accessor("expires_at", { header: "Expires", cell: ({ getValue }) => getValue() ? date(getValue()) : "Never" }),
      columnHelper.accessor("last_used_at", { header: "Last used", cell: ({ getValue }) => date(getValue()) }),
      columnHelper.accessor("created_at", { header: "Created", cell: ({ getValue }) => date(getValue()) }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        meta: { align: "right" },
        cell: ({ row: { original: key } }) =>
          canDelete && (
            <div className="flex justify-end">
              <Button variant="ghost" size="icon" title="Delete" onClick={() => setDeleting(key)}>
                <Trash2 />
              </Button>
            </div>
          ),
      }),
    ],
    [canDelete],
  );

  return (
    <PageShell
      title="API keys"
      subtitle="Keys for scripts and other systems. A key acts as its user, with the roles of that user."
      actions={
        canCreate && (
          <Button onClick={() => setCreating(true)}>
            <Plus /> New API key
          </Button>
        )
      }
    >
      {keysQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-md" />
          ))}
        </div>
      ) : keys.length === 0 ? (
        <EmptyState
          icon={KeySquare}
          title="No API keys"
          description="Create a key, then send it as Authorization: Bearer <key>."
        />
      ) : (
        <DataTable columns={columns} data={keys} getRowId={(key) => key.id} pageSize={10} />
      )}

      <CreateAPIKeyDialog open={creating} onOpenChange={setCreating} />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title={`Delete ${deleting?.name}?`}
        description="Requests with this key fail at once. This cannot be undone."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </PageShell>
  );
}

/** After create, the dialog shows the key. The api never returns it again. */
function CreateAPIKeyDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const queryClient = useQueryClient();
  const [userId, setUserId] = React.useState("");
  const [name, setName] = React.useState("");
  const [expires, setExpires] = React.useState("");

  const usersQuery = useQuery({
    queryKey: ["users", "active"],
    queryFn: () => AdminService.listUsers({ is_active: true, limit: 100 }),
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: () =>
      AdminService.createAPIKey({
        user_id: userId,
        name,
        // the key works until the end of the chosen day
        expires_at: expires ? new Date(`${expires}T23:59:59`).toISOString() : undefined,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["api-keys"] }),
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't create the API key", description: apiErrorMessage(error) }),
  });
  const created = mutation.data;

  React.useEffect(() => {
    if (!open) return;
    setUserId("");
    setName("");
    setExpires("");
    mutation.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{created ? "Copy the API key" : "New API key"}</DialogTitle>
          <DialogDescription>
            {created
              ? "You see this key only now. Save it in a safe place."
              : "The key acts as the chosen user. Use a service account for an integration."}
          </DialogDescription>
        </DialogHeader>

        {created ? (
          <>
            <div className="flex gap-2">
              <Input readOnly aria-label="API key" value={created.key} className="font-mono" onFocus={(e) => e.target.select()} />
              <Button
                type="button"
                variant="outline"
                size="icon"
                title="Copy"
                onClick={() => navigator.clipboard.writeText(created.key).then(() => toast({ title: "Copied" }))}
              >
                <Copy />
              </Button>
            </div>
            <DialogFooter className="mt-2">
              <Button onClick={() => onOpenChange(false)}>Done</Button>
            </DialogFooter>
          </>
        ) : (
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault();
              mutation.mutate();
            }}
          >
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="api_key_user">User</Label>
              <Select value={userId} onValueChange={setUserId} required>
                <SelectTrigger id="api_key_user">
                  <SelectValue placeholder="Choose a user" />
                </SelectTrigger>
                <SelectContent>
                  {usersQuery.data?.entries.map((user) => (
                    <SelectItem key={user.id} value={user.id}>
                      {user.username}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="api_key_name">Name</Label>
              <Input id="api_key_name" required placeholder="For example: SIEM integration" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="api_key_expires">Expires (optional)</Label>
              <Input id="api_key_expires" type="date" value={expires} onChange={(e) => setExpires(e.target.value)} />
            </div>
            <DialogFooter className="mt-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending || !userId} showLoader={mutation.isPending}>
                Create
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
