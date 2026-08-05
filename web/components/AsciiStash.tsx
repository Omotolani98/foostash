"use client";

import { useEffect, useRef } from "react";
import { ENV_KEYS } from "@/lib/content";

const COL_W = 7.2; // IBM Plex Mono advance at 12px
const ROW_H = 14;
const TICK_MS = 90;
const MAX_TOKENS = 12;
const BOX_W = 46;

type Token = { text: string; col: number; row: number; v: number };

/**
 * Decorative hero background: env vars rain down as plaintext, flip to ▒
 * ciphertext past the halfway mark, and land in a drawn "stash" box that
 * fills with ▓.
 *
 * Painted by writing textContent on two <pre> refs rather than through React
 * state — this repaints ~11×/sec and reconciling a full-viewport string that
 * often would dominate the main thread for something purely decorative.
 * Pauses when the tab is hidden or the hero scrolls out of view, and renders
 * a single static frame under prefers-reduced-motion.
 */
export function AsciiStash() {
  const wrapRef = useRef<HTMLDivElement>(null);
  const tokensRef = useRef<HTMLPreElement>(null);
  const frameRef = useRef<HTMLPreElement>(null);
  const fillRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const wrap = wrapRef.current;
    const tokensEl = tokensRef.current;
    const frameEl = frameRef.current;
    const fillEl = fillRef.current;
    if (!wrap || !tokensEl || !frameEl || !fillEl) return;

    let cols = 0;
    let rows = 0;
    let tokens: Token[] = [];
    let fill = 0;
    let frame = 0;

    const measure = () => {
      cols = Math.max(24, Math.floor(wrap.clientWidth / COL_W));
      rows = Math.max(24, Math.floor(wrap.clientHeight / ROW_H));
    };

    const blank = () =>
      Array.from({ length: rows }, () => new Array<string>(cols).fill(" "));

    const put = (grid: string[][], r: number, c: number, s: string) => {
      if (r < 0 || r >= rows) return;
      for (let i = 0; i < s.length; i++) {
        const x = c + i;
        if (x >= 0 && x < cols) grid[r][x] = s[i];
      }
    };

    const spawn = () => {
      const text = ENV_KEYS[Math.floor(Math.random() * ENV_KEYS.length)];
      const span = Math.max(1, cols - text.length - 4);
      tokens.push({
        text,
        col: 2 + Math.floor(Math.random() * span),
        row: -1,
        v: 0.28 + Math.random() * 0.4,
      });
    };

    const advance = () => {
      const lid = rows - 9;
      frame += 1;
      if (frame % 5 === 0 && tokens.length < MAX_TOKENS) spawn();
      for (const t of tokens) t.row += t.v;

      const landed = tokens.filter((t) => t.row >= lid).length;
      if (landed) {
        const bw = Math.min(BOX_W, cols - 8);
        const capacity = (bw - 4) * (rows - 3 - (lid + 1) + 1);
        fill = (fill + landed * 3) % (capacity + 12);
        tokens = tokens.filter((t) => t.row < lid);
      }
    };

    const paint = () => {
      const lid = rows - 9;
      const gTokens = blank();
      const gFrame = blank();
      const gFill = blank();

      for (const t of tokens) {
        const r = Math.floor(t.row);
        if (r < 0 || r >= lid) continue;
        // Past the halfway point the value is already ciphertext.
        const encrypted = t.row > lid * 0.55;
        put(gTokens, r, t.col, encrypted ? "▒".repeat(t.text.length) : t.text);
      }

      const bw = Math.min(BOX_W, cols - 8);
      const x0 = Math.floor((cols - bw) / 2);
      const label = " stash ";
      const side = Math.floor((bw - 2 - label.length) / 2);

      put(
        gFrame,
        lid,
        x0,
        "┌" +
          "─".repeat(side) +
          label +
          "─".repeat(Math.max(0, bw - 2 - side - label.length)) +
          "┐",
      );

      const innerTop = lid + 1;
      const innerBottom = rows - 3;
      for (let r = innerTop; r <= innerBottom; r++) {
        put(gFrame, r, x0, "│");
        put(gFrame, r, x0 + bw - 1, "│");
      }
      put(gFrame, rows - 2, x0, "└" + "─".repeat(bw - 2) + "┘");

      const iw = bw - 4;
      let left = Math.min(fill, iw * (innerBottom - innerTop + 1));
      for (let r = innerBottom; r >= innerTop && left > 0; r--) {
        const n = Math.min(left, iw);
        put(gFill, r, x0 + 2, "▓".repeat(n));
        left -= n;
      }

      const flatten = (g: string[][]) => g.map((r) => r.join("")).join("\n");
      tokensEl.textContent = flatten(gTokens);
      frameEl.textContent = flatten(gFrame);
      fillEl.textContent = flatten(gFill);
    };

    measure();

    const ro = new ResizeObserver(() => {
      measure();
      paint();
    });
    ro.observe(wrap);

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      // One composed frame: a partly-filled stash, a few values in flight.
      for (let i = 0; i < 6; i++) {
        spawn();
        tokens[i].row = Math.random() * (rows - 12);
      }
      fill = Math.floor((BOX_W - 4) * 2.5);
      paint();
      return () => ro.disconnect();
    }

    let timer: number | undefined;
    let onScreen = true;

    const sync = () => {
      const shouldRun = onScreen && !document.hidden;
      if (shouldRun && timer === undefined) {
        timer = window.setInterval(() => {
          advance();
          paint();
        }, TICK_MS);
      } else if (!shouldRun && timer !== undefined) {
        clearInterval(timer);
        timer = undefined;
      }
    };

    const io = new IntersectionObserver(
      ([entry]) => {
        onScreen = entry.isIntersecting;
        sync();
      },
      { threshold: 0 },
    );
    io.observe(wrap);

    document.addEventListener("visibilitychange", sync);
    paint();
    sync();

    return () => {
      if (timer !== undefined) clearInterval(timer);
      document.removeEventListener("visibilitychange", sync);
      io.disconnect();
      ro.disconnect();
    };
  }, []);

  return (
    <div className="ascii" ref={wrapRef} aria-hidden="true">
      <pre className="ascii__tokens" ref={tokensRef} />
      <pre className="ascii__fill" ref={fillRef} />
      <pre className="ascii__frame" ref={frameRef} />
    </div>
  );
}
