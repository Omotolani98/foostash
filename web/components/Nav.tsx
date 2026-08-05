"use client";

import { useEffect, useState } from "react";
import { NAV_LINKS, REPO } from "@/lib/content";
import { GithubMark } from "./GithubMark";

/**
 * N5 floating pill. The design nests this inside the hero panel, which clips
 * `position: sticky` — the nav would scroll away and take its own section
 * anchors with it. Fixed at page level instead, so it stays reachable.
 */
export function Nav() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia("(min-width: 60rem)");
    const sync = () => mq.matches && setOpen(false);
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  return (
    <header className="pill">
      <div className="pill__bar">
        <a className="pill__brand" href="#top">
          <span className="brand__bracket">[</span>
          <span>foostash</span>
          <span className="brand__bracket">]</span>
        </a>

        <nav className="pill__links" aria-label="Sections">
          {NAV_LINKS.map((link) => (
            <a key={link.href} className="pill__link" href={link.href}>
              {link.label}
            </a>
          ))}
        </nav>

        <div className="pill__right">
          <a
            className="pill__gh"
            href={REPO}
            target="_blank"
            rel="noreferrer"
            title="GitHub"
          >
            <GithubMark size={16} />
            <span className="sr-only">foostash on GitHub</span>
          </a>

          <button
            type="button"
            className="pill__toggle"
            aria-expanded={open}
            aria-controls="pill-sheet"
            onClick={() => setOpen((v) => !v)}
          >
            <span className="sr-only">{open ? "Close menu" : "Open menu"}</span>
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              aria-hidden="true"
            >
              {open ? (
                <path d="M3.5 3.5l9 9M12.5 3.5l-9 9" />
              ) : (
                <path d="M2 4.5h12M2 8h12M2 11.5h12" />
              )}
            </svg>
          </button>
        </div>
      </div>

      <div className="pill__sheet" id="pill-sheet" data-open={open}>
        <ul>
          {NAV_LINKS.map((link) => (
            <li key={link.href}>
              <a href={link.href} onClick={() => setOpen(false)}>
                {link.label}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </header>
  );
}
