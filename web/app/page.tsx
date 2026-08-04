import { Fragment } from "react";
import { Nav } from "@/components/Nav";
import { Terminal } from "@/components/Terminal";
import { InstallStrip } from "@/components/InstallStrip";
import {
  ARCH_EDGES,
  ARCH_NODES,
  FEATURES,
  REPO,
  SECURITY_ROWS,
  SELFHOST_STEPS,
  VERSION,
} from "@/lib/content";

export default function Home() {
  return (
    <div id="top">
      <Nav />

      <main>
        {/* HERO */}
        <section className="hero">
          <div className="hero__copy">
            <div className="hero__badges">
              <span className="badge">open source · MIT</span>
              <span className="badge">{VERSION}</span>
              <span className="badge badge--accent">single Go binary</span>
            </div>

            <h1 className="hero__title">
              Secrets that never leave your machine unencrypted.
            </h1>

            <p className="hero__lede">
              Encrypted, versioned secrets and environment manager. SSH-key
              auth, a self-hostable server, and zero knowledge by design — the
              server never sees plaintext. Retire your <code>.env</code> files
              and Slack-pasted credentials.
            </p>

            <div className="hero__actions">
              <InstallStrip />
              <div className="hero__links">
                <a href="#selfhost">→ self-host in one command</a>
                <a href="/docs">→ read the docs</a>
              </div>
            </div>
          </div>

          <Terminal />
        </section>

        {/* FEATURES */}
        <section id="features" className="section">
          <div className="section__inner">
            <div className="section__head">
              <span className="section__num">01</span>
              <h2 className="section__title">What you get</h2>
            </div>
            <div className="grid">
              {FEATURES.map((f) => (
                <div className="cell" key={f.title}>
                  <span className="cell__title">{f.title}</span>
                  <span className="cell__body">{f.body}</span>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ARCHITECTURE */}
        <section id="architecture" className="section">
          <div className="section__inner">
            <div className="section__head">
              <span className="section__num">02</span>
              <h2 className="section__title">Architecture</h2>
            </div>
            <p className="section__lede">
              One CLI, one server, one Postgres. Nothing else to run.
            </p>
            <div className="arch">
              {ARCH_NODES.map((node, i) => (
                <Fragment key={node.name}>
                  {i > 0 && (
                    <div className="arch__edge" aria-hidden="true">
                      <span className="arch__over">{ARCH_EDGES[i - 1].over}</span>
                      <span className="arch__arrow" />
                      <span className="arch__under">
                        {ARCH_EDGES[i - 1].under}
                      </span>
                    </div>
                  )}
                  <div className="arch__node">
                    <span className="arch__name">{node.name}</span>
                    <span className="arch__where">{node.where}</span>
                    <span className="arch__note">
                      {node.note.map((n, j) => (
                        <Fragment key={n}>
                          {j > 0 && <br />}
                          {n}
                        </Fragment>
                      ))}
                    </span>
                  </div>
                </Fragment>
              ))}
            </div>
          </div>
        </section>

        {/* SECURITY */}
        <section id="security" className="section">
          <div className="section__inner">
            <div className="section__head">
              <span className="section__num">03</span>
              <h2 className="section__title">Security model</h2>
            </div>
            <p className="section__lede">
              Short enough to actually read. The full model is in{" "}
              <a
                href={`${REPO}/blob/dev/docs/security.md`}
                target="_blank"
                rel="noreferrer"
              >
                docs/security.md
              </a>
              .
            </p>
            <div className="spec">
              {SECURITY_ROWS.map((row) => (
                <div className="spec__row" key={row.term}>
                  <span className="spec__term">{row.term}</span>
                  <span className="spec__body">{row.body}</span>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* SELF-HOST */}
        <section id="selfhost" className="section">
          <div className="section__inner">
            <div className="section__head">
              <span className="section__num">04</span>
              <h2 className="section__title">Self-host in three steps</h2>
            </div>
            <p className="section__lede">
              Your server, your Postgres, your keys. No cloud dependency, no
              pricing page.
            </p>
            <div className="steps">
              {SELFHOST_STEPS.map((step, i) => (
                <div className="step" key={step.label}>
                  <div className="step__label">
                    <span className="step__n">{i + 1}</span> · {step.label}
                  </div>
                  <pre>
                    {step.lines.map((line, j) => (
                      <Fragment key={j}>
                        {j > 0 && "\n"}
                        {"prompt" in line && line.prompt && (
                          <span className="step__sigil">$ </span>
                        )}
                        {"comment" in line && line.comment ? (
                          <span className="step__comment">{line.text}</span>
                        ) : (
                          line.text
                        )}
                      </Fragment>
                    ))}
                  </pre>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>

      {/* FOOTER */}
      <footer className="foot">
        <div className="foot__inner">
          <span>
            <span className="brand__bracket">[</span>foostash
            <span className="brand__bracket">]</span> · MIT · written in Go
          </span>
          <div className="foot__links">
            <a href={REPO} target="_blank" rel="noreferrer">
              github
            </a>
            <a href="/docs">docs</a>
            <a href={`${REPO}/releases`} target="_blank" rel="noreferrer">
              releases
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
