import * as React from "react";
import { Link } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { CalendarClock, Pencil, Plus, Trash2 } from "lucide-react";

import ScheduleService from "@/services/schedules/schedules";
import { Schedule } from "@/services/schedules/schedules.schema";
import PlaybookService from "@/services/playbooks/playbooks";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import { DataTable, EmptyState, PageShell } from "@/components/soar";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ConfirmDialog } from "@/pages/settings/components/ConfirmDialog";

const columnHelper = createColumnHelper<Schedule>();
const date = (value: string | null) => (value ? new Date(value).toLocaleString() : "-");

const timezones = ["UTC", ...Intl.supportedValuesOf("timeZone").filter((tz) => tz !== "UTC")];

/** Why a schedule does not run now, or null when it runs. */
export function scheduleNote(schedule: Schedule): string | null {
  if (schedule.playbook_deleted) return "Playbook deleted";
  if (schedule.playbook_trigger_type !== "manual") return "Playbook is not manual";
  return null;
}

export default function SchedulesPage() {
  const queryClient = useQueryClient();
  const canCreate = usePermission("schedules", "create");
  const canUpdate = usePermission("schedules", "update");
  const canDelete = usePermission("schedules", "delete");
  const [editing, setEditing] = React.useState<Schedule | "new" | null>(null);
  const [deleting, setDeleting] = React.useState<Schedule | null>(null);

  const schedulesQuery = useQuery({ queryKey: ["schedules"], queryFn: () => ScheduleService.list() });
  const schedules = schedulesQuery.data ?? [];
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["schedules"] });

  const toggle = useMutation({
    mutationFn: (schedule: Schedule) => ScheduleService.update(schedule.id, { enabled: !schedule.enabled }),
    onSuccess: refresh,
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't change the schedule", description: apiErrorMessage(error) }),
  });

  const remove = useMutation({
    mutationFn: () => ScheduleService.remove(deleting!.id),
    onSuccess: () => {
      toast({ variant: "success", title: "Schedule deleted" });
      refresh();
      setDeleting(null);
    },
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't delete the schedule", description: apiErrorMessage(error) }),
  });

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Name",
        cell: ({ row: { original: schedule } }) => (
          <div>
            <div className="font-semibold">{schedule.name}</div>
            {scheduleNote(schedule) && <div className="text-[12.5px] text-rose-text">{scheduleNote(schedule)}</div>}
          </div>
        ),
      }),
      columnHelper.accessor("playbook_name", {
        header: "Playbook",
        cell: ({ row: { original: schedule } }) => (
          <Link className="hover:underline" to={`/playbooks/${schedule.playbook_id}`}>
            {schedule.playbook_name}
          </Link>
        ),
      }),
      columnHelper.accessor("cron", {
        header: "Timetable",
        cell: ({ row: { original: schedule } }) => (
          <div>
            <div className="font-mono text-[13px]">{schedule.cron}</div>
            <div className="text-[12.5px] text-ink-faint">{schedule.timezone}</div>
          </div>
        ),
      }),
      columnHelper.accessor("next_run_at", {
        header: "Next run",
        cell: ({ row: { original: schedule } }) => (schedule.enabled ? date(schedule.next_run_at) : "-"),
      }),
      columnHelper.accessor("last_run_at", { header: "Last run", cell: ({ getValue }) => date(getValue()) }),
      columnHelper.accessor("enabled", {
        header: "Enabled",
        cell: ({ row: { original: schedule } }) => (
          <Button
            variant={schedule.enabled ? "default" : "outline"}
            size="sm"
            role="switch"
            aria-checked={schedule.enabled}
            aria-label={`Enable ${schedule.name}`}
            disabled={!canUpdate || toggle.isPending}
            onClick={() => toggle.mutate(schedule)}
          >
            {schedule.enabled ? "On" : "Off"}
          </Button>
        ),
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        meta: { align: "right" },
        cell: ({ row: { original: schedule } }) => (
          <div className="flex justify-end">
            {canUpdate && (
              <Button variant="ghost" size="icon" title="Edit" onClick={() => setEditing(schedule)}>
                <Pencil />
              </Button>
            )}
            {canDelete && (
              <Button variant="ghost" size="icon" title="Delete" onClick={() => setDeleting(schedule)}>
                <Trash2 />
              </Button>
            )}
          </div>
        ),
      }),
    ],
    [canUpdate, canDelete, toggle],
  );

  return (
    <PageShell
      title="Schedules"
      subtitle="Run a manual playbook on a timetable. A missed time after a downtime does not run."
      actions={
        canCreate && (
          <Button onClick={() => setEditing("new")}>
            <Plus /> New schedule
          </Button>
        )
      }
    >
      {schedulesQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-md" />
          ))}
        </div>
      ) : schedules.length === 0 ? (
        <EmptyState
          icon={CalendarClock}
          title="No schedules"
          description="Create a schedule to run a playbook at a fixed time."
        />
      ) : (
        <DataTable columns={columns} data={schedules} getRowId={(schedule) => schedule.id} pageSize={10} />
      )}

      <ScheduleDialog editing={editing} onClose={() => setEditing(null)} onSaved={refresh} />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title={`Delete ${deleting?.name}?`}
        description="The playbook stays. The runs of this schedule stay in the run history."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </PageShell>
  );
}

function ScheduleDialog({
  editing,
  onClose,
  onSaved,
}: {
  editing: Schedule | "new" | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const open = editing !== null;
  const schedule = editing === "new" ? null : editing;
  const [name, setName] = React.useState("");
  const [playbookId, setPlaybookId] = React.useState("");
  const [cron, setCron] = React.useState("");
  const [timezone, setTimezone] = React.useState("UTC");
  const [parameters, setParameters] = React.useState("{}");

  React.useEffect(() => {
    if (!open) return;
    setName(schedule?.name ?? "");
    setPlaybookId(schedule?.playbook_id ?? "");
    setCron(schedule?.cron ?? "");
    setTimezone(schedule?.timezone ?? "UTC");
    setParameters(JSON.stringify(schedule?.parameters ?? {}, null, 2));
  }, [open, schedule]);

  const playbooksQuery = useQuery({
    queryKey: ["playbooks", "manual"],
    queryFn: () => PlaybookService.getPlaybooks(0, 100, { trigger_type: "manual" }),
    enabled: open,
  });

  // the api computes the times, so the UI needs no cron library
  const previewQuery = useQuery({
    queryKey: ["schedule-preview", cron, timezone],
    queryFn: () => ScheduleService.preview(cron.trim(), timezone),
    enabled: open && cron.trim() !== "",
    retry: false,
  });

  const parsedParameters = React.useMemo((): Record<string, unknown> | null => {
    try {
      const value = JSON.parse(parameters || "{}");
      return value && typeof value === "object" && !Array.isArray(value) ? value : null;
    } catch {
      return null;
    }
  }, [parameters]);

  const mutation = useMutation({
    mutationFn: () => {
      const body = { name, cron: cron.trim(), timezone, parameters: parsedParameters! };
      return schedule
        ? ScheduleService.update(schedule.id, body)
        : ScheduleService.create({ ...body, playbook_id: playbookId, enabled: true });
    },
    onSuccess: () => {
      toast({ variant: "success", title: schedule ? "Schedule saved" : "Schedule created" });
      onSaved();
      onClose();
    },
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't save the schedule", description: apiErrorMessage(error) }),
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{schedule ? "Edit schedule" : "New schedule"}</DialogTitle>
          <DialogDescription>
            Use 5 fields: minute, hour, day of month, month, day of week. Example: 0 2 * * * runs at 02:00 each day.
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
            <Label htmlFor="schedule_name">Name</Label>
            <Input id="schedule_name" required value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="schedule_playbook">Playbook</Label>
            <Select value={playbookId} onValueChange={setPlaybookId} disabled={schedule !== null}>
              <SelectTrigger id="schedule_playbook">
                <SelectValue placeholder="Choose a manual playbook" />
              </SelectTrigger>
              <SelectContent>
                {playbooksQuery.data?.entries.map((playbook) => (
                  <SelectItem key={playbook.id} value={playbook.id}>
                    {playbook.name}
                  </SelectItem>
                ))}
                {schedule && !playbooksQuery.data?.entries.some((p) => p.id === schedule.playbook_id) && (
                  <SelectItem value={schedule.playbook_id}>{schedule.playbook_name}</SelectItem>
                )}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule_cron">Cron</Label>
              <Input
                id="schedule_cron"
                required
                className="font-mono"
                placeholder="0 2 * * *"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule_timezone">Time zone</Label>
              <Select value={timezone} onValueChange={setTimezone}>
                <SelectTrigger id="schedule_timezone">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {timezones.map((tz) => (
                    <SelectItem key={tz} value={tz}>
                      {tz}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          {cron.trim() !== "" && (
            <div className="text-[13px]" aria-live="polite">
              {previewQuery.isError ? (
                <span className="text-rose-text">{apiErrorMessage(previewQuery.error)}</span>
              ) : (
                <>
                  <div className="text-ink-faint">Next runs</div>
                  {previewQuery.data?.map((time) => <div key={time}>{new Date(time).toLocaleString()}</div>)}
                </>
              )}
            </div>
          )}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="schedule_parameters">Parameters (JSON)</Label>
            <Textarea
              id="schedule_parameters"
              className="font-mono"
              rows={4}
              value={parameters}
              onChange={(e) => setParameters(e.target.value)}
            />
            {parsedParameters === null && <span className="text-[13px] text-rose-text">Parameters must be a JSON object.</span>}
          </div>
          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={mutation.isPending || parsedParameters === null || (!schedule && !playbookId)}
              showLoader={mutation.isPending}
            >
              {schedule ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
