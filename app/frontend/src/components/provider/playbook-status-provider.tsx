"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getPlaybookWsUrl } from "@/settings/urls";
import { Edges, TaskHistory } from "@/services/playbooks/playbooks.schema";

type StatusEvent = {
  event: "task_status" | "playbook_status";
  data: Record<string, unknown> & {
    id?: string;
    playbook_id?: string;
    playbook_history_id?: string;
    task_id?: string;
  };
};

type TaskHistoryQueryData = { tasks: TaskHistory[]; edges: Edges[] };

const MAX_BACKOFF = 5000;

const PlaybookStatusContext = createContext<{ connected: boolean }>({
  connected: false,
});

export const usePlaybookStatusConnection = () =>
  useContext(PlaybookStatusContext);

/**
 * One websocket for the whole app. The hub is a fanout, so a single connection
 * receives every run's events; we fan them into the right React Query caches:
 *  - task_status     → merge the row into that run's task-history cache so the
 *                      flow's status rings update live (no refetch).
 *  - playbook_status → the run finished; refresh the execution lists so they
 *                      flip to success/failed too.
 */
const PlaybookStatusProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
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

        if (
          msg.event === "task_status" &&
          msg.data.playbook_history_id &&
          msg.data.task_id
        ) {
          const key = [`worfklow-task-history-${msg.data.playbook_history_id}`];
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
          if (msg.data.playbook_id) {
            queryClient.invalidateQueries({
              queryKey: [`workflow-history-${msg.data.playbook_id}`],
            });
          }
          queryClient.invalidateQueries({ queryKey: ["playbooks-history-all"] });
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
  }, [queryClient]);

  return (
    <PlaybookStatusContext.Provider value={{ connected }}>
      {children}
    </PlaybookStatusContext.Provider>
  );
};

export default PlaybookStatusProvider;
