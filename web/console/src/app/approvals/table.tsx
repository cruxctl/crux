"use client";

import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/features/layouts/table";
import type { ApprovalRecord } from "@/features/api/types";

const columns: ColumnDef<ApprovalRecord, unknown>[] = [
  {
    header: "ID",
    cell: ({ row }) => (
      <span className="mono">{row.original.id.slice(0, 12)}…</span>
    ),
  },
  { header: "Tool call", accessorKey: "tool_call" },
  { header: "Policy", accessorKey: "policy_id" },
  {
    header: "Approvers",
    cell: ({ row }) => row.original.approvers?.join(", ") || "—",
  },
  {
    header: "Status",
    accessorKey: "status",
    cell: ({ row }) => {
      const status = row.original.status;
      const color =
        status === "pending"
          ? "text-yellow-600 dark:text-yellow-400"
          : status === "granted"
          ? "text-green-600 dark:text-green-400"
          : status === "denied"
          ? "text-red-600 dark:text-red-400"
          : "text-neutral-500";
      return <span className={color}>{status}</span>;
    },
  },
  { header: "Created", accessorKey: "created_at" },
];

export function ApprovalsTable({ approvals }: { approvals: ApprovalRecord[] }) {
  return <DataTable data={approvals} columns={columns} />;
}
