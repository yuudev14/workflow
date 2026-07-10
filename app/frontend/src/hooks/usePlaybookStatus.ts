import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getPlaybookWsUrl } from "@/settings/urls";
import { Edges, TaskHistory } from "@/services/playbooks/playbooks.schema";

type StatusEvent = {
  event: "task_status" | "playbook_status";
  // task_status carries a full task_history row; playbook_status a history row.
  data: Record<string, unknown> & {
    playbook_history_id?: string;
    task_id?: string;
  };
};

type TaskHistoryQueryData = { tasks: TaskHistory[]; edges: Edges[] };

const MAX_BACKOFF = 5000;

/**
 * Subscribe to /ws/playbook and keep the task-history query for a single run
 * live. The hub is a fanout (every client gets every event), so we filter by
 * playbook_history_id and merge each task_status row into the cached query by
 * task_id — no refetch, the flow re-renders with fresh status rings.
 */
const usePlaybookStatus = (playbookHistoryId?: string) => {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);

  // keep the latest id in a ref so onmessage always filters against it without
  // reopening the socket.
  const historyIdRef = useRef(playbookHistoryId);
  historyIdRef.current = playbookHistoryId;

  useEffect(() => {
    if (!playbookHistoryId) return;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let backoff = 1000;
    let closed = false; // set on unmount so we stop reconnecting

    const connect = () => {
      ws = new WebSocket(getPlaybookWsUrl());

      ws.onopen = () => {
        setConnected(true);
        backoff = 1000;
      };

      ws.onmessage = (e) => {
        let msg: StatusEvent;
        try {
          msg = JSON.parse(e.data);
        } catch {
          return;
        }
        const currentId = historyIdRef.current;
        if (!currentId || msg.data?.playbook_history_id !== currentId) return;

        if (msg.event === "task_status" && msg.data.task_id) {
          const key = [`worfklow-task-history-${currentId}`];
          queryClient.setQueryData<TaskHistoryQueryData>(key, (old) => {
            if (!old) return old;
            return {
              ...old,
              tasks: old.tasks.map((t) =>
                t.task_id === msg.data.task_id
                  ? ({ ...t, ...msg.data } as TaskHistory)
                  : t
              ),
            };
          });
        } else if (msg.event === "playbook_status") {
          // overall run status changed — refresh the history list views.
          queryClient.invalidateQueries({ queryKey: ["playbooks-history"] });
        }
      };

      ws.onclose = () => {
        setConnected(false);
        if (closed) return;
        reconnectTimer = setTimeout(connect, backoff);
        backoff = Math.min(backoff * 2, MAX_BACKOFF);
      };

      ws.onerror = () => {
        ws?.close();
      };
    };

    connect();

    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, [playbookHistoryId, queryClient]);

  return { connected };
};

export default usePlaybookStatus;
