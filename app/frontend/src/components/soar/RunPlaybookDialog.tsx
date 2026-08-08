"use client";

import * as React from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Plus, X, Zap } from "lucide-react";

import PlaybookService from "@/services/playbooks/playbooks";
import AlertService from "@/services/alerts/alerts";
import IncidentService from "@/services/incidents/incidents";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

type Param = { key: string; value: string };

/**
 * Runs a playbook against selected alerts or incidents.
 *
 * Only record ids go over the wire; the server hydrates the rows. Templates
 * then reach them as `var.input.records[...]` and the analyst's key/values as
 * `var.input.parameters`.
 *
 * One run covers N records rather than N runs - a playbook author loops inside
 * a code snippet if per-record work is wanted.
 */
export function RunPlaybookDialog({
  open,
  onOpenChange,
  moduleType,
  recordIds,
  onLaunched,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moduleType: "alert" | "incident";
  recordIds: string[];
  onLaunched?: () => void;
}) {
  const [playbookId, setPlaybookId] = React.useState("");
  const [params, setParams] = React.useState<Param[]>([]);

  React.useEffect(() => {
    if (open) {
      setPlaybookId("");
      setParams([]);
    }
  }, [open]);

  const playbooksQuery = useQuery({
    queryKey: ["playbooks", "runnable"],
    queryFn: () => PlaybookService.getPlaybooks(0, 200),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });

  const run = useMutation({
    mutationFn: () => {
      const parameters = Object.fromEntries(
        params.filter((p) => p.key.trim()).map((p) => [p.key.trim(), p.value]),
      );
      const payload = {
        playbook_id: playbookId,
        record_ids: recordIds,
        ...(Object.keys(parameters).length ? { parameters } : {}),
      };
      return moduleType === "alert"
        ? AlertService.runPlaybook(payload)
        : IncidentService.runPlaybook(payload);
    },
    onSuccess: () => {
      onOpenChange(false);
      onLaunched?.();
    },
  });

  // jinja2 raises on records[0] against an empty list while nunjucks renders
  // "", so a zero-record run is a trap rather than a no-op.
  const tooMany = recordIds.length > 100;
  const canRun = !!playbookId && recordIds.length > 0 && !tooMany && !run.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Run playbook</DialogTitle>
          <DialogDescription>
            One run against {recordIds.length} {moduleType}
            {recordIds.length === 1 ? "" : "s"}. The playbook reads them as{" "}
            <code className="mono">var.input.records</code>.
          </DialogDescription>
        </DialogHeader>

        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (canRun) run.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label>Playbook</Label>
            <Select value={playbookId} onValueChange={setPlaybookId}>
              <SelectTrigger>
                <SelectValue placeholder="Pick a playbook…" />
              </SelectTrigger>
              <SelectContent>
                {(playbooksQuery.data?.entries ?? []).map((p) => (
                  <SelectItem key={p.id} value={p.id} className="text-[13px]">
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>
              Parameters <span className="font-normal text-ink-faint">optional</span>
            </Label>
            {params.map((p, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  placeholder="key"
                  value={p.key}
                  onChange={(e) =>
                    setParams((prev) =>
                      prev.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)),
                    )
                  }
                />
                <Input
                  placeholder="value"
                  value={p.value}
                  onChange={(e) =>
                    setParams((prev) =>
                      prev.map((x, j) => (j === i ? { ...x, value: e.target.value } : x)),
                    )
                  }
                />
                <button
                  type="button"
                  onClick={() => setParams((prev) => prev.filter((_, j) => j !== i))}
                  className="text-ink-faint hover:text-rose-text"
                  aria-label="Remove parameter"
                >
                  <X className="size-4" />
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => setParams((prev) => [...prev, { key: "", value: "" }])}
              className="inline-flex w-fit items-center gap-1.5 text-[12.5px] font-semibold text-signal-text hover:brightness-110"
            >
              <Plus className="size-3.5" /> Add parameter
            </button>
          </div>

          {tooMany && (
            <p className="text-[12.5px] text-rose-text">
              {recordIds.length} selected - a run is capped at 100 records.
            </p>
          )}
          {run.isError && (
            <p className="text-[12.5px] text-rose-text">
              The run was rejected. Check that you hold {moduleType}s:execute.
            </p>
          )}

          <DialogFooter>
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="rounded-sm border border-line-strong px-3.5 py-2 text-[13px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!canRun}
              className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3.5 py-2 text-[13px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              <Zap className="size-3.5" />
              {run.isPending ? "Starting…" : "Run"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
