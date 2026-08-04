"use client";

import { useEffect, useRef, useState } from "react";
import { INSTALL_CMD } from "@/lib/content";

export function InstallStrip() {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => () => clearTimeout(timer.current), []);

  async function copy() {
    try {
      await navigator.clipboard.writeText(INSTALL_CMD);
    } catch {
      // Clipboard blocked (insecure origin, denied permission). The command
      // is selectable text either way — don't claim a copy that didn't happen.
      return;
    }
    setCopied(true);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div className="install">
      <span className="install__sigil" aria-hidden="true">
        $
      </span>
      <code className="install__cmd">{INSTALL_CMD}</code>
      <button
        type="button"
        className="install__copy"
        data-copied={copied}
        onClick={copy}
      >
        {copied ? "copied ✓" : "copy"}
        <span className="sr-only"> install command</span>
      </button>
    </div>
  );
}
