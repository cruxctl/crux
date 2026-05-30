import { listAOSEvents } from "@/features/api/aos";

export const dynamic = "force-dynamic";

export default async function AOSPage() {
  const events = await listAOSEvents();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">AOS Events</h1>
        <span className="pageCount">
          {events.length} event{events.length !== 1 ? "s" : ""}
        </span>
      </div>
      <div className="borderBox">
        <table className="table">
          <thead >
            <tr>
              <th >Event ID</th>
              <th >Type</th>
              <th >Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {events.map((e) => (
              <tr key={e.event_id} >
                <td className="mono">{e.event_id}</td>
                <td >{e.event_type}</td>
                <td >
                  {new Date(e.timestamp).toLocaleString()}
                </td>
              </tr>
            ))}
            {events.length === 0 && (
              <tr>
                <td colSpan={3} className="empty">
                  No events found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
