"use client";

import * as React from "react";
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
import type { Severity } from "@/services/incidents/incidents.schema";

const SEVERITIES: Severity[] = ["critical", "high", "medium", "low"];

export function NewIncidentDialog({
  open,
  onOpenChange,
  pending,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending?: boolean;
  onSubmit: (payload: { title: string; severity: Severity }) => void;
}) {
  const [title, setTitle] = React.useState("");
  const [severity, setSeverity] = React.useState<Severity>("high");

  React.useEffect(() => {
    if (open) {
      setTitle("");
      setSeverity("high");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New incident</DialogTitle>
          <DialogDescription>
            Opens an empty case. Link alerts to it from the alert queue, or escalate an alert
            instead to get the link for free.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit({ title: title.trim(), severity });
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="incident_title">Title</Label>
            <Input
              id="incident_title"
              value={title}
              required
              placeholder="Ransomware staging on FIN-WS-114"
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="incident_severity">Severity</Label>
            <Select value={severity} onValueChange={(v) => setSeverity(v as Severity)}>
              <SelectTrigger id="incident_severity">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SEVERITIES.map((s) => (
                  <SelectItem key={s} value={s}>
                    {s[0].toUpperCase() + s.slice(1)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <button
              type="submit"
              disabled={pending || !title.trim()}
              className="rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              {pending ? "Creating…" : "Create incident"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
