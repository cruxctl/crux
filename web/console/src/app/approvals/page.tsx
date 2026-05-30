import { listApprovals } from "@/features/api/approvals";
import { ApprovalsTable } from "./table";

export const dynamic = "force-dynamic";

export default async function ApprovalsPage() {
  const approvals = await listApprovals();
  return (
    <div className="page">
      <div className="pageHeader">
        <h1 className="pageTitle">Approvals</h1>
        <span className="pageCount">
          {approvals.length} approval{approvals.length !== 1 ? "s" : ""}
        </span>
      </div>
      <ApprovalsTable approvals={approvals} />
    </div>
  );
}
