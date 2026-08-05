"use client";

import { useEffect, useRef } from "react";
import { ENV_KEYS } from "@/lib/content";

const TICK_MS = 90;
const MAX_TOKENS = 12;
const BOX_W = 46;
const FALLBACK_ADVANCE = 7.2;
const FALLBACK_ROW = 14;

/**
 * Box-drawing and block glyphs only line up if they advance by exactly the
 * same width as a space in the font that actually renders them. When they
 * don't, horizontal runs drift and the box stops closing, so we verify at
 * runtime and drop to pure ASCII if the check fails.
 */
const UNICODE_GLYPHS = {
  tl: "┌",
  tr: "┐",
  bl: "└",
  br: "┘",
  h: "─",
  v: "│",
  fill: "▓",
  cipher: "▒",
};

const ASCII_GLYPHS = {
  tl: "+",
  tr: "+",
  bl: "+",
  br: "+",
  h: "-",
  v: "|",
  fill: "#",
  cipher: "*",
};

type Token = { text: string; col: number; row: number; v: number };

/**
 * Decorative hero background: env vars rain down as plaintext, flip to
 * ciphertext past the halfway mark, and land in a drawn "stash" box that
 * fills from the bottom.
 *
 * Painted by writing textContent on three <pre> refs rather than through
 * React state — this repaints ~11x/sec and reconciling a full-viewport
 * string that often would dominate the main thread for something purely
 * decorative. Pauses when the tab is hidden or the hero scrolls out of
 * view, and renders a single static frame under prefers-reduced-motion.
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
    let advance = FALLBACK_ADVANCE;
    let rowH = FALLBACK_ROW;
    let glyphs = UNICODE_GLYPHS;
    let tokens: Token[] = [];
    let fill = 0;
    let frame = 0;

    /** Width of one character cell, measured in the font that really loaded. */
    const widthOf = (char: string, style: CSSStyleDeclaration) => {
      const probe = document.createElement("span");
      probe.style.cssText =
        "position:absolute;visibility:hidden;white-space:pre;padding:0;border:0";
      probe.style.fontFamily = style.fontFamily;
      probe.style.fontSize = style.fontSize;
      probe.textContent = char.repeat(50);
      wrap.appendChild(probe);
      const w = probe.getBoundingClientRect().width / 50;
      probe.remove();
      return w;
    };

    const measure = () => {
      const style = getComputedStyle(tokensEl);

      const base = widthOf("0", style);
      advance = base > 0 ? base : FALLBACK_ADVANCE;

      // Every glyph the art uses must match the base cell, or the grid skews.
      const monospaced = Object.values(UNICODE_GLYPHS).every(
        (g) => Math.abs(widthOf(g, style) - advance) < 0.5,
      );
      glyphs = monospaced ? UNICODE_GLYPHS : ASCII_GLYPHS;

      const parsedRow = parseFloat(style.lineHeight);
      rowH = Number.isFinite(parsedRow) && parsedRow > 0 ? parsedRow : FALLBACK_ROW;

      cols = Math.max(24, Math.floor(wrap.clientWidth / advance));
      rows = Math.max(24, Math.floor(wrap.clientHeight / rowH));
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

    const advanceFrame = () => {
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
        put(
          gTokens,
          r,
          t.col,
          encrypted ? glyphs.cipher.repeat(t.text.length) : t.text,
        );
      }

      const bw = Math.min(BOX_W, cols - 8);
      const x0 = Math.floor((cols - bw) / 2);
      const label = " stash ";
      const side = Math.floor((bw - 2 - label.length) / 2);

      put(
        gFrame,
        lid,
        x0,
        glyphs.tl +
          glyphs.h.repeat(side) +
          label +
          glyphs.h.repeat(Math.max(0, bw - 2 - side - label.length)) +
          glyphs.tr,
      );

      const innerTop = lid + 1;
      const innerBottom = rows - 3;
      for (let r = innerTop; r <= innerBottom; r++) {
        put(gFrame, r, x0, glyphs.v);
        put(gFrame, r, x0 + bw - 1, glyphs.v);
      }
      put(
        gFrame,
        rows - 2,
        x0,
        glyphs.bl + glyphs.h.repeat(bw - 2) + glyphs.br,
      );

      const iw = bw - 4;
      let left = Math.min(fill, iw * (innerBottom - innerTop + 1));
      for (let r = innerBottom; r >= innerTop && left > 0; r--) {
        const n = Math.min(left, iw);
        put(gFill, r, x0 + 2, glyphs.fill.repeat(n));
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

    // Metrics can change once the webfont swaps in.
    if (document.fonts?.ready) {
      document.fonts.ready.then(() => {
        measure();
        paint();
      });
    }

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
          advanceFrame();
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
