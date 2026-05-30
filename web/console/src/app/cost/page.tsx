import { getCosts } from "@/features/api/cost";

export const dynamic = "force-dynamic";

export default async function CostPage() {
  const costs = await getCosts();

  const byAgent: Record<string, number> = {};
  for (const c of costs) {
    byAgent[c.agent_id] = (byAgent[c.agent_id] ?? 0) + c.usd;
  }
  const rows = Object.entries(byAgent).sort((a, b) => b[1] - a[1]);
  const total = rows.reduce((s, [, v]) => s + v, 0);

  return (
    <div className="detailPage">
      <h1 className="pageTitle mb4">Cost</h1>
      <p className="pageCount mb6">
        Estimated cost per agent derived from token counts.
      </p>
      <div className="rounded border border-neutral-200 dark:border-neutral-800 overflow-hidden">
        <table className="min-w-full text-sm">
          <thead >
            <tr>
              <th >
                Agent
              </th>
              <th >
                Cost (USD)
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td
                  colSpan={2}
                  className="empty"
                >
                  No cost data recorded yet.
                </td>
              </tr>
            ) : (
              rows.map(([k, v]) => (
                <tr
                  key={k}
                  
                >
                  <td >{k}</td>
                  <td >${v.toFixed(4)}</td>
                </tr>
              ))
            )}
            {rows.length > 0 && (
              <tr >
                <td >Total</td>
                <td >${total.toFixed(4)}</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
