import { getMetrics } from "@/features/api/metrics";

export const dynamic = "force-dynamic";

export default async function MetricsPage() {
  const metrics = await getMetrics();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Metrics</h1>
        <span className="pageCount">
          {metrics.length} metric{metrics.length !== 1 ? "s" : ""}
        </span>
      </div>
      <div className="cardGrid">
        {metrics.map((m) => (
          <div key={m.name} className="card">
            <div className="pageCount">{m.name}</div>
            <div className="cardValue">{m.value}</div>
            <div className="cardLabel">
              {new Date(m.timestamp).toLocaleString()}
            </div>
          </div>
        ))}
        {metrics.length === 0 && (
          <div className="empty">
            No metrics available
          </div>
        )}
      </div>
    </div>
  );
}
