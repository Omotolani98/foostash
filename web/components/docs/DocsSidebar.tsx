"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { DOC_GROUPS } from "@/lib/docs";

type IndexEntry = {
  slug: string;
  title: string;
  group: string;
  haystack: string;
};

/**
 * Grouped page nav with a client-side filter. Receives a precomputed search
 * index from the server layout so the full docs content never ships twice.
 */
export function DocsSidebar({ index }: { index: IndexEntry[] }) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  const q = query.trim().toLowerCase();
  const visible = q ? index.filter((p) => p.haystack.includes(q)) : index;

  const nav = (
    <nav className="docs-nav" aria-label="Documentation">
      <div className="docs-search">
        <span aria-hidden="true">⌕</span>
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="search docs"
          aria-label="Search docs"
        />
      </div>

      {DOC_GROUPS.map((group) => {
        const items = visible.filter((p) => p.group === group);
        if (items.length === 0) return null;
        return (
          <div className="docs-nav__group" key={group}>
            <span className="docs-nav__label">{group}</span>
            {items.map((p) => {
              const href = `/docs/${p.slug}`;
              const active = pathname === href;
              return (
                <Link
                  key={p.slug}
                  href={href}
                  className="docs-nav__item"
                  data-active={active}
                  aria-current={active ? "page" : undefined}
                  onClick={() => {
                    setOpen(false);
                    setQuery("");
                  }}
                >
                  {p.title}
                </Link>
              );
            })}
          </div>
        );
      })}

      {visible.length === 0 && (
        <p className="docs-nav__empty">no pages match “{query}”</p>
      )}
    </nav>
  );

  return (
    <>
      {/* Mobile: a disclosure bar under the top bar; desktop: the fixed rail. */}
      <button
        type="button"
        className="docs-nav-toggle"
        aria-expanded={open}
        aria-controls="docs-nav-panel"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? "× close" : "≡ docs menu"}
      </button>
      <div className="docs-rail" id="docs-nav-panel" data-open={open}>
        {nav}
      </div>
    </>
  );
}
