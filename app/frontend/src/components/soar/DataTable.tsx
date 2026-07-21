"use client";

import * as React from "react";
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { cn } from "@/lib/utils";
import { PaginationBar } from "./PaginationBar";

declare module "@tanstack/react-table" {
  interface ColumnMeta<TData, TValue> {
    align?: "left" | "right";
  }
}

export function DataTable<TData>({
  columns,
  data,
  getRowId,
  className,
  pageSize,
}: {
  columns: ColumnDef<TData, any>[];
  data: TData[];
  getRowId?: (row: TData) => string;
  className?: string;
  /** When set, the table paginates client-side and shows a footer. */
  pageSize?: number;
}) {
  const paginated = pageSize != null;
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: paginated ? getPaginationRowModel() : undefined,
    initialState: paginated ? { pagination: { pageSize } } : undefined,
    getRowId,
  });

  return (
    <div className={cn("overflow-hidden rounded-md border border-line", className)}>
      <div className="overflow-x-auto">
        <table className="w-full text-[13.5px]">
          <thead className="bg-paper-sunken text-[12px] uppercase tracking-wide text-ink-soft">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className={cn(
                      "px-3 py-2.5 font-semibold",
                      header.column.columnDef.meta?.align === "right" ? "text-right" : "text-left"
                    )}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => (
              <tr
                key={row.id}
                className="border-t border-line transition-colors hover:bg-paper-sunken/60"
              >
                {row.getVisibleCells().map((cell) => (
                  <td
                    key={cell.id}
                    className={cn(
                      "px-3 py-2.5 align-middle",
                      cell.column.columnDef.meta?.align === "right" ? "text-right" : undefined
                    )}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {paginated && (
        <PaginationBar
          page={table.getState().pagination.pageIndex}
          pageCount={table.getPageCount()}
          total={data.length}
          pageSize={pageSize}
          onPrev={() => table.previousPage()}
          onNext={() => table.nextPage()}
          className="border-t border-line bg-paper-sunken/30 px-3 py-2"
        />
      )}
    </div>
  );
}
