import { listMachines } from "@/features/api/machines";

export const dynamic = "force-dynamic";

export default async function MachinesPage() {
  const machines = await listMachines();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Machines</h1>
        <span className="pageCount">
          {machines.length} machine{machines.length !== 1 ? "s" : ""}
        </span>
      </div>
      <div className="borderBox">
        <table className="table">
          <thead >
            <tr>
              <th >ID</th>
              <th >Name</th>
              <th >Status</th>
              <th >OS</th>
              <th >Arch</th>
            </tr>
          </thead>
          <tbody>
            {machines.map((m) => (
              <tr key={m.id} >
                <td className="mono">{m.id}</td>
                <td >{m.name || "—"}</td>
                <td >
                  <span className="badge badgeGreen">
                    {m.status}
                  </span>
                </td>
                <td >{m.os || "—"}</td>
                <td >{m.arch || "—"}</td>
              </tr>
            ))}
            {machines.length === 0 && (
              <tr>
                <td colSpan={5} className="empty">
                  No machines enrolled
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
