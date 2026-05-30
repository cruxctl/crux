"use client";

import Link from "next/link";
import {
  Activity,
  Server,
  Database,
  ShieldCheck,
  BadgeDollarSign,
  Terminal,
  ScrollText,
  Radio,
  Gauge,
  Cpu,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import styles from "./shell.module.css";

const NAV: Array<{ href: string; label: string; icon: LucideIcon }> = [
  { href: "/fleet", label: "Fleet", icon: Server },
  { href: "/sessions", label: "Sessions", icon: Terminal },
  { href: "/executions", label: "Executions", icon: Activity },
  { href: "/mcp", label: "MCP Registry", icon: Database },
  { href: "/policies", label: "Policies", icon: ScrollText },
  { href: "/approvals", label: "Approvals", icon: ShieldCheck },
  { href: "/aos", label: "Events", icon: Radio },
  { href: "/cost", label: "Cost", icon: BadgeDollarSign },
  { href: "/metrics", label: "Metrics", icon: Gauge },
  { href: "/machines", label: "Machines", icon: Cpu },
];

export function Shell({ children }: { children: React.ReactNode }) {
  const toggleTheme = () => {
    const html = document.documentElement;
    const current = html.getAttribute("data-theme");
    if (current === "dark") {
      html.removeAttribute("data-theme");
    } else {
      html.setAttribute("data-theme", "dark");
    }
  };

  return (
    <div className={styles.shell}>
      <aside className={styles.sidebar}>
        <div className={styles.brand}>
          Crux <span>Console</span>
        </div>
        <nav className={styles.nav}>
          {NAV.map(({ href, label, icon: Icon }) => (
            <Link key={href} href={href as any} className={styles.navLink}>
              <Icon className={styles.navIcon} />
              {label}
            </Link>
          ))}
        </nav>
        <button className={styles.themeToggle} onClick={toggleTheme}>
          Toggle theme
        </button>
        <div className={styles.version}>V0.2 — wired</div>
      </aside>
      <main className={styles.main}>{children}</main>
    </div>
  );
}
