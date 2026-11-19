import type { McpCatalog } from "@cal/api-client";
import { KeyRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { usePlanner } from "../stores/planner";

export function AgentPage() {
  const api = usePlanner((state) => state.api);
  const [catalog, setCatalog] = useState<McpCatalog | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let live = true;
    api
      .mcpTools()
      .then((next) => {
        if (live) setCatalog(next);
      })
      .catch(() => {
        if (live) setFailed(true);
      });
    return () => {
      live = false;
    };
  }, [api]);

  const count = catalog?.tools.length ?? 0;

  return (
    <>
      <PageHeader title="Agents" sub={catalog ? `${count} tools` : "MCP"}>
        <Link className="btn btn-secondary" to="/settings#set-mcp">
          <KeyRound size={14} /> API token
        </Link>
      </PageHeader>
      <div className="page-scroll" data-testid="agent-page">
        <section className="panel agent-panel">
          <p className="panel-note">
            Assistants call <code>POST /api/mcp</code> with the API token. This list is the live server catalog
            {catalog ? ` (${catalog.protocol})` : ""}. The token stays in Settings.
          </p>
          {failed && <p className="panel-empty">The tool catalog did not load.</p>}
          {!failed && !catalog && <p className="panel-empty">Loading tools…</p>}
          {catalog && (
            <ul className="agent-tools">
              {catalog.tools.map((tool) => (
                <li key={tool.name} className="agent-tool">
                  <code>{tool.name}</code>
                  <span>{tool.description}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </>
  );
}
