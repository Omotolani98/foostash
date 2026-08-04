import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { DOC_PAGES, anchorFor, type DocBlock } from "@/lib/docs";

type Params = { slug: string };

export function generateStaticParams(): Params[] {
  return DOC_PAGES.map((p) => ({ slug: p.slug }));
}

export const dynamicParams = false;

export async function generateMetadata({
  params,
}: {
  params: Promise<Params>;
}): Promise<Metadata> {
  const { slug } = await params;
  const page = DOC_PAGES.find((p) => p.slug === slug);
  if (!page) return {};
  return { title: page.title, description: page.lede };
}

function Block({ block }: { block: DocBlock }) {
  switch (block.t) {
    case "h2":
      return <h2 id={anchorFor(block.text)}>{block.text}</h2>;
    case "p":
      return <p>{block.text}</p>;
    case "code":
      return <CodeBlock lang={block.lang} code={block.code} />;
    case "note":
      return (
        <aside className="doc-note">
          <span className="doc-note__tag">note</span>
          <span>{block.text}</span>
        </aside>
      );
    case "ul":
      return (
        <ul className="doc-list">
          {block.items.map((item, i) => (
            <li key={i}>
              <span className="doc-list__dot" aria-hidden="true">
                ·
              </span>
              <span>
                {item.term && <strong>{item.term}</strong>}
                {item.text}
              </span>
            </li>
          ))}
        </ul>
      );
    case "table":
      return (
        <div className="doc-table">
          <table>
            <thead>
              <tr>
                {block.head.map((h) => (
                  <th key={h} scope="col">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {block.rows.map((row, i) => (
                <tr key={i}>
                  {row.map((cell, j) => (
                    <td key={j} data-lead={j === 0}>
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
  }
}

export default async function DocPage({
  params,
}: {
  params: Promise<Params>;
}) {
  const { slug } = await params;
  const idx = DOC_PAGES.findIndex((p) => p.slug === slug);
  if (idx === -1) notFound();

  const page = DOC_PAGES[idx];
  const prev = DOC_PAGES[idx - 1];
  const next = DOC_PAGES[idx + 1];
  const toc = page.blocks.filter((b) => b.t === "h2");

  return (
    <>
      <main className="docs-main">
        <div className="docs-crumbs" aria-hidden="true">
          <span>docs</span>
          <span>/</span>
          <span className="docs-crumbs__here">{page.slug}</span>
        </div>
        <h1>{page.title}</h1>
        <p className="docs-lede">{page.lede}</p>

        {page.blocks.map((block, i) => (
          <Block block={block} key={i} />
        ))}

        <nav className="docs-pager" aria-label="Adjacent pages">
          {prev ? (
            <Link href={`/docs/${prev.slug}`} className="docs-pager__card">
              <span className="docs-pager__dir">← previous</span>
              <span className="docs-pager__title">{prev.title}</span>
            </Link>
          ) : (
            <span />
          )}
          {next && (
            <Link
              href={`/docs/${next.slug}`}
              className="docs-pager__card docs-pager__card--next"
            >
              <span className="docs-pager__dir">next →</span>
              <span className="docs-pager__title">{next.title}</span>
            </Link>
          )}
        </nav>
      </main>

      <aside className="docs-toc">
        <span className="docs-toc__label">on this page</span>
        {toc.length > 0 && (
          <div className="docs-toc__links">
            {toc.map((b) => (
              <a href={`#${anchorFor(b.text)}`} key={b.text}>
                {b.text}
              </a>
            ))}
          </div>
        )}
        <a
          className="docs-toc__edit"
          href={page.source}
          target="_blank"
          rel="noreferrer"
        >
          edit on github ↗
        </a>
      </aside>
    </>
  );
}
