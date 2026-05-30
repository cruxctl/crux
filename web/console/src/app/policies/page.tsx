import { listPolicies } from "@/features/api/policies";

export const dynamic = "force-dynamic";

export default async function PoliciesPage() {
  const policies = await listPolicies();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Policies</h1>
        <span className="pageCount">
          {policies.length} polic{policies.length !== 1 ? "ies" : "y"}
        </span>
      </div>
      <div className="borderBox">
        <table className="table">
          <thead >
            <tr>
              <th >ID</th>
              <th >Rules</th>
            </tr>
          </thead>
          <tbody>
            {policies.map((p) => (
              <tr key={p.metadata.id} >
                <td className="mono">{p.metadata.id}</td>
                <td >{p.rules.length} rules</td>
              </tr>
            ))}
            {policies.length === 0 && (
              <tr>
                <td colSpan={2} className="empty">
                  No policies found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
