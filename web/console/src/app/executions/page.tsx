import { listExecutions } from "@/features/api/executions";
import { ExecutionsTable } from "./table";

export const dynamic = "force-dynamic";

export default async function ExecutionsPage() {
  const executions = await listExecutions();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Executions</h1>
        <span className="pageCount">
          {executions.length} execution{executions.length !== 1 ? "s" : ""}
        </span>
      </div>
      <ExecutionsTable executions={executions} />
    </div>
  );
}
