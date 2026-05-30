import { getMCPServer } from "@/features/api/mcp";
import { notFound } from "next/navigation";
import Link from "next/link";

export const dynamic = "force-dynamic";

export default async function MCPServerDetail({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;
  const server = await getMCPServer(name);
  if (!server) notFound();

  return (
    <div className="detailPage">
      <div className="mb4">
        <Link
          href="/mcp"
          className="link"
        >
          ← MCP Registry
        </Link>
      </div>
      <h1 className="pageTitle mb6">{server.name}</h1>
      <dl className="detailGrid dl">
        <dt className="dt">
          ID
        </dt>
        <dd className="mono">{server.id}</dd>
        <dt className="dt">
          URL
        </dt>
        <dd className="mono">{server.url ?? "—"}</dd>
        <dt className="dt">
          Version
        </dt>
        <dd>{server.version ?? "—"}</dd>
        <dt className="dt">
          Status
        </dt>
        <dd>{server.status}</dd>
      </dl>
      {server.tools && server.tools.length > 0 && (
        <div className="mb6">
          <h2 className="detailSectionTitle">Tools</h2>
          <ul >
            {server.tools.map((tool) => (
              <li
                key={tool}
                className="mono"
              >
                {tool}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
