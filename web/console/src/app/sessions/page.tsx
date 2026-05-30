import { listSessions } from "@/features/api/sessions";

export const dynamic = "force-dynamic";

export default async function SessionsPage() {
  const sessions = await listSessions();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Sessions</h1>
        <span className="pageCount">
          {sessions.length} session{sessions.length !== 1 ? "s" : ""}
        </span>
      </div>
      <div className="borderBox">
        <table className="table">
          <thead >
            <tr>
              <th >ID</th>
              <th >Agent</th>
              <th >Status</th>
              <th >Started</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map((s) => (
              <tr key={s.id} >
                <td className="mono">{s.id}</td>
                <td >{s.agent_id}</td>
                <td >
                  <StatusBadge status={s.status} />
                </td>
                <td >
                  {new Date(s.started_at).toLocaleString()}
                </td>
              </tr>
            ))}
            {sessions.length === 0 && (
              <tr>
                <td colSpan={4} className="empty">
                  No sessions found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    running: "badgeGreen",
    completed: "badgeBlue",
    failed: "badgeRed",
    canceled: "",
    created: "badgeYellow",
    continued: "",
  };
  return (
    <span className={`badge ${map[status] || ""}`}>
      {status}
    </span>
  );
}
