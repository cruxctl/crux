"use client";

import Link from "next/link";
import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/features/layouts/table";
import type { Execution } from "@/features/api/types";

const columns: ColumnDef<Execution, unknown>[] = [
  {
    header: "ID",
    cell: ({ row }) => (
      <Link
        href={`/executions/${row.original.id}` as `/executions/${string}`}
        className="link mono"
      >
        {row.original.id.slice(0, 12)}…
      </Link>
    ),
  },
  { header: "Agent", accessorKey: "agentName" },
  {
    header: "Status",
    accessorKey: "status",
    cell: ({ row }) => {
      const status = row.original.status;
      const color =
        status === "succeeded"
          ? "text-green-600 dark:text-green-400"
          : status === "failed"
          ? "text-red-600 dark:text-red-400"
          : status === "running"
          ? "text-blue-600 dark:text-blue-400"
          : "text-neutral-500";
      return <span className={color}>{status}</span>;
    },
  },
  {
    header: "Prompt",
    cell: ({ row }) => (
      <span className="mono">
        {row.original.prompt}
      </span>
    ),
  },
  { header: "Queued", accessorKey: "queuedAt" },
];

export function ExecutionsTable({ executions }: { executions: Execution[] }) {
  return <DataTable data={executions} columns={columns} />;
}
