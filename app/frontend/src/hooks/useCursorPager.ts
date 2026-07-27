"use client";

import * as React from "react";

export const PAGE_SIZES = [25, 50, 100, 200] as const;

/**
 * Paging state for a keyset-cursor list endpoint.
 *
 * The API has no `offset` - it takes an opaque `cursor` and hands back
 * `next_cursor`, which is what keeps deep pages fast and correct while rows are
 * being inserted. That buys forward/back paging but not random access: to go
 * back we replay the cursor we used on the way in, so this keeps a stack of the
 * cursors visited. There is deliberately no "jump to page 7" - the API cannot
 * answer it, and faking it with a client-side offset would silently skip rows.
 *
 * `resetKey` should be whatever identifies the current filter. Changing it drops
 * the stack, because a cursor is only meaningful for the query that produced it.
 */
export function useCursorPager(resetKey: string, defaultLimit = 25) {
  const [limit, setLimitState] = React.useState<number>(defaultLimit);
  // One entry per visited page; undefined is page 1 (no cursor).
  const [stack, setStack] = React.useState<(string | undefined)[]>([undefined]);

  React.useEffect(() => {
    setStack([undefined]);
  }, [resetKey]);

  const goNext = React.useCallback((nextCursor?: string) => {
    if (nextCursor) setStack((s) => [...s, nextCursor]);
  }, []);

  const goPrev = React.useCallback(() => {
    setStack((s) => (s.length > 1 ? s.slice(0, -1) : s));
  }, []);

  const setLimit = React.useCallback((n: number) => {
    setLimitState(n);
    setStack([undefined]);
  }, []);

  return {
    cursor: stack[stack.length - 1],
    limit,
    setLimit,
    /** Zero-based, for computing the displayed row range. */
    page: stack.length - 1,
    canPrev: stack.length > 1,
    goNext,
    goPrev,
  };
}
