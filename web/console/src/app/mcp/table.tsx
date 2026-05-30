"use client";

import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/features/layouts/table";
import type { MCPServer } from "@/features/api/types";

const columns: ColumnDef<MCPServer, unknown>[] = [
  { header: "Name", accessorKey: "name" },
  { header: "URL", accessorKey: "url" },
  { header: "Version", accessorKey: "version" },
  { header: "Status", accessorKey: "status" },
  {
    header: "Tools",
    cell: ({ row }) => row.original.tools?.length || 0,
  },
];

export function MCPTable({ servers }: { servers: MCPServer[] }) {
  return <DataTable data={servers} columns={columns} />;
}
