"use client";

import { useEffect, useState } from "react";
import { NAV_LINKS, REPO } from "@/lib/content";
import { GithubMark } from "./GithubMark";

export function Nav() {
  const [scrolled, setScrolled] = useState(false);
  const [open, setOpen] = useState(false);

  // rAF-throttled so the handler never runs more than once per frame.
  useEffect(() => {
    let frame = 0;
    const onScroll = () => {
      if (frame) return;
      frame = requestAnimationFrame(() => {
        setScrolled(window.scrollY > 8);
        frame = 0;
      });
    };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      if (frame) cancelAnimationFrame(frame);
    };
  }, []);

  // The sheet is a mobile affordance; if the viewport grows past the
  // breakpoint while it is open, drop it so state matches what is visible.
  useEffect(() => {
    const mq = window.matchMedia("(min-width: 60rem)");
    const sync = () => mq.matches && setOpen(false);
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);

  return (
    <header className="nav" data-scrolled={scrolled}>
      <div className="nav__inner">
        <a className="nav__brand" href="#top">
          <span className="brand__bracket">[</span>
          <span>foostash</span>
          <span className="brand__bracket">]</span>
        </a>

        <nav className="nav__center" aria-label="Sections">
          {NAV_LINKS.map((link) => (
            <a
              key={link.href}
              className="nav__link"
              href={link.href}
              {...("external" in link && link.external
                ? { target: "_blank", rel: "noreferrer" }
                : {})}
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="nav__right">
          <a
            className="nav__cta"
            href={REPO}
            target="_blank"
            rel="noreferrer"
          >
            <GithubMark />
            <span className="nav__cta-label">star on github</span>
            <span className="sr-only">star foostash on GitHub</span>
          </a>

          <button
            type="button"
            className="nav__toggle"
            aria-expanded={open}
            aria-controls="nav-sheet"
            onClick={() => setOpen((v) => !v)}
          >
            <span className="sr-only">
              {open ? "Close menu" : "Open menu"}
            </span>
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

      <div className="nav__sheet" id="nav-sheet" data-open={open}>
        <ul>
          {NAV_LINKS.map((link) => (
            <li key={link.href}>
              <a
                href={link.href}
                onClick={() => setOpen(false)}
                {...("external" in link && link.external
                  ? { target: "_blank", rel: "noreferrer" }
                  : {})}
              >
                {link.label}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </header>
  );
}
