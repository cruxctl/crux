import { listMCPServers } from "@/features/api/mcp";
import { MCPTable } from "./table";

export const dynamic = "force-dynamic";

export default async function MCPPage() {
  const servers = await listMCPServers();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">MCP Registry</h1>
        <span className="pageCount">
          {servers.length} server{servers.length !== 1 ? "s" : ""}
        </span>
      </div>
      <MCPTable servers={servers} />
    </div>
  );
}
