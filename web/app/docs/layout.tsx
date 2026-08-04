import type { Metadata } from "next";
import Link from "next/link";
import { DocsSidebar } from "@/components/docs/DocsSidebar";
import { searchIndex } from "@/lib/docs";
import { REPO } from "@/lib/content";

export const metadata: Metadata = {
  title: {
    template: "%s — foostash docs",
    default: "foostash docs",
  },
};

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="docs">
      <header className="docs-bar">
        <div className="docs-bar__crumb">
          <Link href="/" className="nav__brand">
            <span className="brand__bracket">[</span>
            <span>foostash</span>
            <span className="brand__bracket">]</span>
          </Link>
          <span className="docs-bar__sep" aria-hidden="true">
            /
          </span>
          <span className="docs-bar__here">docs</span>
          <span className="docs-bar__branch">dev</span>
        </div>
        <a
          className="docs-bar__gh"
          href={REPO}
          target="_blank"
          rel="noreferrer"
        >
          github ↗
        </a>
      </header>

      <div className="docs-shell">
        <DocsSidebar index={searchIndex()} />
        {children}
      </div>
    </div>
  );
}
