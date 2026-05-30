import { listAgents } from "@/features/api/agents";
import { FleetTable } from "./table";

export const dynamic = "force-dynamic";

export default async function FleetPage() {
  const agents = await listAgents();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Fleet</h1>
        <span className="pageCount">
          {agents.length} agent{agents.length !== 1 ? "s" : ""}
        </span>
      </div>
      <FleetTable agents={agents} />
    </div>
  );
}
