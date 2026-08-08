"use client";

import * as React from "react";
import { Tag, X } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Editable tag list.
 *
 * Each add or remove saves the whole array, because the API takes tags as one
 * `Nullable[[]string]` value rather than a patch of individual entries - there
 * is no add-one endpoint to call. An empty array clears the column; omitting
 * the key entirely is what leaves it alone, which is why the caller must not
 * send `tags` on unrelated edits.
 */
export function TagsField({
  tags,
  canEdit,
  pending,
  onChange,
}: {
  tags: string[];
  canEdit: boolean;
  pending?: boolean;
  onChange: (tags: string[]) => void;
}) {
  const [draft, setDraft] = React.useState("");

  const add = () => {
    const value = draft.trim().replace(/,$/, "");
    if (!value || tags.includes(value)) {
      setDraft("");
      return;
    }
    onChange([...tags, value]);
    setDraft("");
  };

  if (!canEdit) {
    if (tags.length === 0) return <span className="text-[13px] text-ink-faint">No tags</span>;
    return (
      <div className="flex flex-wrap gap-1.5">
        {tags.map((t) => (
          <span
            key={t}
            className="inline-flex items-center gap-1 rounded-sm border border-line bg-paper-sunken px-1.5 py-0.5 text-[12px] text-ink-soft"
          >
            <Tag className="size-3" />
            {t}
          </span>
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {tags.map((t) => (
            <span
              key={t}
              className="inline-flex items-center gap-1 rounded-sm border border-line bg-paper-sunken py-0.5 pl-1.5 pr-1 text-[12px] text-ink-soft"
            >
              <Tag className="size-3" />
              {t}
              <button
                type="button"
                onClick={() => onChange(tags.filter((x) => x !== t))}
                disabled={pending}
                aria-label={`Remove tag ${t}`}
                className="text-ink-faint hover:text-rose-text disabled:opacity-50"
              >
                <X className="size-3" />
              </button>
            </span>
          ))}
        </div>
      )}
      <input
        value={draft}
        disabled={pending}
        onChange={(e) => setDraft(e.target.value)}
        // Enter and comma both commit; blur too, so a typed tag is never lost
        // by clicking away.
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === ",") {
            e.preventDefault();
            add();
          } else if (e.key === "Backspace" && !draft && tags.length) {
            onChange(tags.slice(0, -1));
          }
        }}
        onBlur={add}
        placeholder="Add a tag…"
        className={cn(
          "rounded-sm border border-line-strong bg-background px-2.5 py-1.5 text-[13px] outline-none",
          "placeholder:text-ink-faint focus:border-signal-dot disabled:opacity-50",
        )}
      />
    </div>
  );
}
