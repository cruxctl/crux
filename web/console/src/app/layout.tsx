import type { Metadata } from "next";
import "./globals.css";
import { Shell } from "@/features/layouts/shell";

export const metadata: Metadata = {
  title: "Crux Console",
  description: "Vendor-neutral AI agent control plane",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <Shell>{children}</Shell>
      </body>
    </html>
  );
}
