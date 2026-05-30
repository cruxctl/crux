import Link from "next/link";
import type { Route } from "next";

const CARDS: Array<{ href: Route; title: string; desc: string }> = [
  {
    href: "/fleet",
    title: "Fleet",
    desc: "Managed agents discovered on this control plane.",
  },
  {
    href: "/executions",
    title: "Executions",
    desc: "Recent and active jobs across all agents.",
  },
  {
    href: "/mcp",
    title: "MCP Registry",
    desc: "MCP servers, schema pinning, trust scores.",
  },
  {
    href: "/approvals",
    title: "Approvals",
    desc: "Pending approvals for side-effecting actions.",
  },
  {
    href: "/cost",
    title: "Cost",
    desc: "Per-agent cost rollups (estimated for managed agents).",
  },
];

export default function Home() {
  return (
    <div className="page">
      <h1 className="dashboardTitle">Crux Console</h1>
      <p className="dashboardSubtitle">
        Vendor-neutral AI agent control plane. V0.2 — wired.
      </p>
      <div className="cardGrid">
        {CARDS.map((c) => (
          <Link key={c.href} href={c.href} className="dashboardCard">
            <div className="dashboardCardTitle">{c.title}</div>
            <div className="dashboardCardDesc">{c.desc}</div>
          </Link>
        ))}
      </div>
    </div>
  );
}
