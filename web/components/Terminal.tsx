"use client";

import { useEffect, useState } from "react";
import { TERM_SCRIPT } from "@/lib/content";

const TYPE_MS = 32;
const PAUSE_AFTER_CMD = 420;
const PAUSE_BEFORE_CMD = 700;
const PAUSE_BETWEEN_OUT = 180;

const PROMPT = { cmd: "$", ok: "✓", out: " " } as const;

/**
 * Replays a short foostash session. `cmd` lines type character by character;
 * output lines print whole, the way a real shell does.
 *
 * Under prefers-reduced-motion the whole transcript renders at once — the
 * content is the point, the typing is decoration.
 */
export function Terminal() {
  const [progress, setProgress] = useState<number[]>([]);
  const [runId, setRunId] = useState(0);

  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      setProgress(TERM_SCRIPT.map((line) => line.text.length));
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const wait = (ms: number) =>
      new Promise<void>((resolve) => {
        timer = setTimeout(resolve, ms);
      });

    setProgress([]);

    (async () => {
      const chars: number[] = [];

      for (let i = 0; i < TERM_SCRIPT.length; i++) {
        const line = TERM_SCRIPT[i];
        chars.push(0);

        if (line.kind === "cmd") {
          for (let c = 1; c <= line.text.length; c++) {
            await wait(TYPE_MS);
            if (cancelled) return;
            chars[i] = c;
            setProgress([...chars]);
          }
          await wait(PAUSE_AFTER_CMD);
        } else {
          await wait(60);
          if (cancelled) return;
          chars[i] = line.text.length;
          setProgress([...chars]);
          const next = TERM_SCRIPT[i + 1];
          await wait(
            next?.kind === "cmd" ? PAUSE_BEFORE_CMD : PAUSE_BETWEEN_OUT,
          );
        }

        if (cancelled) return;
      }
    })();

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [runId]);

  return (
    <div className="term">
      <div className="term__bar">
        <span>~/myapp</span>
        <button
          type="button"
          className="term__replay"
          onClick={() => setRunId((n) => n + 1)}
        >
          ↻ replay
        </button>
      </div>

      <div className="term__body" aria-live="off">
        {progress.map((shown, i) => {
          const line = TERM_SCRIPT[i];
          const isLast = i === progress.length - 1;
          const typing = line.kind === "cmd" && shown < line.text.length;
          const showCursor =
            isLast && (typing || i === TERM_SCRIPT.length - 1);

          return (
            <div className="term__line" key={i}>
              <span className={`term__prompt term__prompt--${line.kind}`}>
                {PROMPT[line.kind]}
              </span>
              <span className={`term__text--${line.kind}`}>
                {line.text.slice(0, shown)}
                {showCursor ? <span className="term__cursor" /> : null}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
