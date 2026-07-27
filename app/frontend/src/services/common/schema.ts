export interface EntryResponse<T> {
  entries: T[]
  total: number

}

/**
 * Keyset-paginated list. `next_cursor` is an opaque token - pass it straight
 * back as `?cursor=`; never parse or construct one client-side.
 */
export interface CursorPage<T> {
  entries: T[]
  total: number
  next_cursor?: string
}
