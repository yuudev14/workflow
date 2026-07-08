import * as React from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

export interface StepperStep {
  label: string;
}

export function Stepper({
  steps,
  current,
  className,
}: {
  steps: StepperStep[];
  /** index of the current (in-progress) step; steps before it are done */
  current: number;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center", className)}>
      {steps.map((s, i) => {
        const done = i < current;
        const isCurrent = i === current;
        return (
          <React.Fragment key={s.label}>
            {i > 0 && (
              <span
                className={cn(
                  "mx-2 h-[1.5px] w-[34px]",
                  i <= current ? "bg-moss-dot" : "bg-line-strong"
                )}
              />
            )}
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "flex size-[22px] items-center justify-center rounded-full border-[1.5px] text-[11.5px] font-bold",
                  done && "border-moss-dot bg-moss-dot text-white",
                  isCurrent && "border-signal-dot bg-signal-dot text-white",
                  !done && !isCurrent && "border-line-strong bg-paper-sunken text-ink-faint"
                )}
              >
                {done ? <Check className="size-3" /> : i + 1}
              </span>
              <span
                className={cn(
                  "text-[12.5px] font-semibold",
                  done || isCurrent ? "text-foreground" : "text-ink-faint"
                )}
              >
                {s.label}
              </span>
            </div>
          </React.Fragment>
        );
      })}
    </div>
  );
}
