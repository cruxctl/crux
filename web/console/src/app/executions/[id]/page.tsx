import { getExecution } from "@/features/api/executions";
import { listTraceEvents } from "@/features/api/traces";
import { TraceView } from "@/features/layouts/trace-view";
import { notFound } from "next/navigation";
import Link from "next/link";

export const dynamic = "force-dynamic";

export default async function ExecutionDetail({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const execution = await getExecution(id);
  if (!execution) notFound();
  const events = await listTraceEvents();

  return (
    <div className="page">
      <div className="mb4">
        <Link
          href="/executions"
          className="link"
        >
          ← Executions
        </Link>
      </div>
      <h1 className="text-xl font-semibold mb-2 font-mono">{execution.id}</h1>
      <div className="pageCount mb6">
        <span>{execution.agentName}</span>
        <span>•</span>
        <span>{execution.status}</span>
        {execution.startedAt && (
          <>
            <span>•</span>
            <span>{execution.startedAt}</span>
          </>
        )}
      </div>
      <div className="mb6">
        <h2 className="detailSectionTitle">
          Prompt
        </h2>
        <pre className="codeBlock">
          {execution.prompt}
        </pre>
      </div>
      {execution.stdout && (
        <div className="mb6">
          <h2 className="detailSectionTitle">
            Output
          </h2>
          <pre className="codeBlock">
            {execution.stdout}
          </pre>
        </div>
      )}
      {execution.stderr && (
        <div className="mb6">
          <h2 className="detailSectionTitle" style={{color:"var(--accent-warning)"}}>
            Stderr
          </h2>
          <pre className="codeBlock" style={{borderColor:"var(--accent-warning)",color:"var(--accent-warning)"}}>
            {execution.stderr}
          </pre>
        </div>
      )}
      {execution.error && (
        <div className="mb6">
          <h2 className="detailSectionTitle" style={{color:"var(--accent-danger)"}}>
            Error
          </h2>
          <pre className="codeBlock" style={{borderColor:"var(--accent-danger)",color:"var(--accent-danger)"}}>
            {execution.error}
          </pre>
        </div>
      )}
      <h2 className="detailSectionTitle">
        Trace{" "}
        <span className="pageCount">
          ({events.length} event{events.length !== 1 ? "s" : ""})
        </span>
      </h2>
      <TraceView events={events} />
    </div>
  );
}
