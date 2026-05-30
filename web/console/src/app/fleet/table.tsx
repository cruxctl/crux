"use client";

import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/features/layouts/table";
import type { Agent } from "@/features/api/types";

const columns: ColumnDef<Agent, unknown>[] = [
  { header: "Name", accessorKey: "name" },
  { header: "Description", accessorKey: "description" },
  {
    header: "Status",
    accessorKey: "status",
    cell: ({ row }) => {
      const status = row.original.status;
      const color =
        status === "ready"
          ? "text-green-600 dark:text-green-400"
          : status === "busy"
          ? "text-yellow-600 dark:text-yellow-400"
          : status === "error"
          ? "text-red-600 dark:text-red-400"
          : "text-neutral-400";
      return <span className={color}>{status}</span>;
    },
  },
  {
    header: "Command",
    cell: ({ row }) => row.original.command?.path || "—",
  },
];

export function FleetTable({ agents }: { agents: Agent[] }) {
  return <DataTable data={agents} columns={columns} />;
}
