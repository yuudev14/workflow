import * as React from "react";
import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { CalendarClock } from "lucide-react";

import ScheduleService from "@/services/schedules/schedules";
import { usePermission } from "@/hooks/usePermission";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

/** The schedules of one playbook, with a link to the page that edits them. */
export default function PlaybookSchedules({ playbookId }: { playbookId: string }) {
  const canRead = usePermission("schedules", "read");
  const [open, setOpen] = React.useState(false);
  const query = useQuery({
    queryKey: ["schedules", { playbookId }],
    queryFn: () => ScheduleService.list(playbookId),
    enabled: canRead && open,
  });

  if (!canRead) return null;
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost">
          <CalendarClock /> Schedules
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Schedules</DialogTitle>
          <DialogDescription>The timetables that run this playbook.</DialogDescription>
        </DialogHeader>
        {query.data?.length === 0 && <p className="text-[14px] text-ink-soft">This playbook has no schedule.</p>}
        <ul className="flex flex-col gap-2">
          {query.data?.map((schedule) => (
            <li key={schedule.id} className="rounded-md border border-line px-3 py-2">
              <div className="font-semibold">{schedule.name}</div>
              <div className="text-[13px] text-ink-faint">
                <span className="font-mono">{schedule.cron}</span> · {schedule.timezone} ·{" "}
                {schedule.enabled ? `next ${new Date(schedule.next_run_at).toLocaleString()}` : "off"}
              </div>
            </li>
          ))}
        </ul>
        <Link className="text-[14px] font-semibold hover:underline" to="/schedules">
          Open the Schedules page
        </Link>
      </DialogContent>
    </Dialog>
  );
}
