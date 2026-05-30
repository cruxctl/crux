"use client";

import type { AOSEvent } from "@/features/api/types";

export function TraceView({ events }: { events: AOSEvent[] }) {
  if (events.length === 0) {
    return (
      <p className="pageCount">No trace events recorded.</p>
    );
  }

  return (
    <ol >
      {events.map((e) => (
        <li
          key={e.event_id}
          className="borderBox" style={{padding:"0.75rem",fontSize:"var(--text-sm)"}}
        >
          <div className="pageHeader">
            <span className="font-mono text-xs font-medium text-neutral-800 dark:text-neutral-200">
              {e.event_type}
            </span>
            <span className="pageCount">
              {e.timestamp}
            </span>
          </div>
          {e.trace?.span_id && (
            <div className="pageCount">
              span: {e.trace.span_id}
            </div>
          )}
          {e.payload && (
            <pre className="codeBlock">
              {JSON.stringify(e.payload, null, 2)}
            </pre>
          )}
        </li>
      ))}
    </ol>
  );
}
