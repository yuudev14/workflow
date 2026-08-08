"use client";

import * as React from "react";
import { Pencil, Trash2 } from "lucide-react";
import { useAuth } from "@/components/provider/auth-provider";
import { InitialsAvatar } from "./InitialsAvatar";
import { Panel, PanelTitle } from "./Panel";
import { readableDate, relativeAge } from "@/lib/utils";

/** The subset of an alert_note / incident_note row this panel renders. */
export interface SoarNote {
  id: string;
  author_id?: string | null;
  author_username?: string | null;
  body: string;
  created_at: string;
  updated_at: string;
}

/**
 * Analyst notes. Shared by the alert and incident detail pages - both tables
 * have identical columns.
 *
 * Edit and delete show only for the author, mirroring the server rule: notes are
 * evidence in a post-incident review, so a third party silently rewriting one is
 * refused with a 403. A note whose author was deleted is permanently read-only.
 */
export function NotesPanel({
  notes,
  canWrite,
  pending,
  onAdd,
  onEdit,
  onDelete,
  embedded,
}: {
  notes: SoarNote[];
  canWrite: boolean;
  pending?: boolean;
  onAdd: (body: string) => void;
  onEdit: (id: string, body: string) => void;
  onDelete: (id: string) => void;
  /** Drop the Panel chrome, for when a tab container already provides it. */
  embedded?: boolean;
}) {
  const { user } = useAuth();
  const [draft, setDraft] = React.useState("");
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const [editDraft, setEditDraft] = React.useState("");

  const submit = () => {
    const body = draft.trim();
    if (!body) return;
    onAdd(body);
    setDraft("");
  };

  const commitEdit = () => {
    const body = editDraft.trim();
    if (editingId && body) onEdit(editingId, body);
    setEditingId(null);
  };

  const content = (
    <div className="flex flex-col">
        {notes.length === 0 && (
          <p className="pb-2 text-[13px] text-ink-faint">
            No notes yet. Record what you found, or hand the shift over.
          </p>
        )}

        {notes.map((n) => {
          const mine = !!user && !!n.author_id && user.id === n.author_id;
          const edited = n.updated_at !== n.created_at;
          return (
            <div
              key={n.id}
              className="flex gap-2.5 border-t border-line py-2.5 first:border-t-0 first:pt-0"
            >
              <InitialsAvatar name={n.author_username} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5 text-[13px] font-semibold">
                  {n.author_username ?? "unknown"}
                  <span
                    className="font-medium text-ink-faint"
                    title={readableDate(n.created_at)}
                  >
                    {relativeAge(n.created_at)} ago
                  </span>
                  {edited && (
                    <span className="font-medium text-ink-faint" title={readableDate(n.updated_at)}>
                      · edited
                    </span>
                  )}
                  {mine && canWrite && editingId !== n.id && (
                    <span className="ml-auto flex items-center gap-1.5">
                      <button
                        onClick={() => {
                          setEditingId(n.id);
                          setEditDraft(n.body);
                        }}
                        className="text-ink-faint hover:text-foreground"
                        aria-label="Edit note"
                      >
                        <Pencil className="size-3.5" />
                      </button>
                      <button
                        onClick={() => onDelete(n.id)}
                        disabled={pending}
                        className="text-ink-faint hover:text-rose-text"
                        aria-label="Delete note"
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </span>
                  )}
                </div>

                {editingId === n.id ? (
                  <div className="mt-1.5 flex flex-col gap-1.5">
                    <textarea
                      value={editDraft}
                      onChange={(e) => setEditDraft(e.target.value)}
                      rows={3}
                      className="w-full resize-y rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13px] outline-none focus:border-signal-dot"
                    />
                    <div className="flex gap-2">
                      <button
                        onClick={commitEdit}
                        disabled={pending || !editDraft.trim()}
                        className="rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingId(null)}
                        className="rounded-sm border border-line-strong px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="mt-0.5 whitespace-pre-wrap text-[13px] leading-relaxed text-ink-soft">
                    {n.body}
                  </div>
                )}
              </div>
            </div>
          );
        })}

        {canWrite && (
          <div className="mt-2 flex flex-col gap-1.5 border-t border-line pt-2.5">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Add a note…"
              rows={2}
              className="w-full resize-y rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13px] outline-none placeholder:text-ink-faint focus:border-signal-dot"
            />
            <button
              onClick={submit}
              disabled={pending || !draft.trim()}
              className="w-fit rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              {pending ? "Saving…" : "Add note"}
            </button>
          </div>
        )}
    </div>
  );

  if (embedded) return content;

  return (
    <Panel>
      <PanelTitle aside={notes.length > 0 ? `${notes.length}` : undefined}>Notes</PanelTitle>
      {content}
    </Panel>
  );
}
