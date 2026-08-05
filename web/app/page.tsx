import { Fragment } from "react";
import { Nav } from "@/components/Nav";
import { Terminal } from "@/components/Terminal";
import { InstallStrip } from "@/components/InstallStrip";
import { AsciiStash } from "@/components/AsciiStash";
import {
  ARCH_NODES,
  FEATURES,
  REPO,
  SECURITY_ROWS,
  SELFHOST_STEPS,
  STATS,
} from "@/lib/content";

export default function Home() {
  return (
    <div id="top">
      <Nav />

      <main>
        {/* HERO */}
        <section className="hero">
          <div className="hero__panel">
            <AsciiStash />

            <div className="hero__center">
              <h1 className="hero__title">
                Secrets that never leave your machine
              </h1>
              <p className="hero__lede">
                Foostash — encrypted, versioned secrets for teams that
                self-host. SSH-key auth, zero knowledge by design, one Go
                binary.
              </p>

              <div className="hero__cta">
                <a className="btn btn--solid" href="/docs">
                  Get started
                </a>
                <a
                  className="btn btn--ghost"
                  href={REPO}
                  target="_blank"
                  rel="noreferrer"
                >
                  Github
                </a>
              </div>

              <InstallStrip />
            </div>

            <dl className="hero__stats">
              {STATS.map((s) => (
                <div className="hero__stat" key={s.label}>
                  <dt>{s.label}</dt>
                  <dd>{s.value}</dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* TERMINAL */}
        <section className="termwrap" aria-label="Terminal demo">
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
              {ARCH_NODES.map((node) => (
                <div className="arch__card" key={node.name}>
                  <div className="arch__edge">
                    <span className="arch__step">{node.step}</span>
                    <span>{node.edge}</span>
                  </div>
                  <div className="arch__body">
                    <span className="arch__name">{node.name}</span>
                    <span className="arch__where">{node.where}</span>
                    <span className="arch__note">
                      {node.note.map((n, i) => (
                        <Fragment key={n}>
                          {i > 0 && <br />}
                          {n}
                        </Fragment>
                      ))}
                    </span>
                  </div>
                </div>
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
