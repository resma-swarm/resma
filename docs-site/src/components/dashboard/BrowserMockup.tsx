// T5 (uiux): Browser chrome mockup com screenshot do dashboard RESMA.
//
// Mostra o produto em ação dentro de um browser frame custom (Tailwind),
// inspirado em Komodo/Coolify. Não usa react-chrome-mockup — frame custom
// dá controle total sobre o visual dark premium e os tokens RESMA.
//
// Estrutura:
//   chrome bar -> 3 dots (red/yellow/green) + URL bar + refresh icon
//   viewport   -> screenshot (PNG/SVG) com loading="lazy"
//
// Responsividade: o container usa w-full e a imagem h-auto, escalando
// naturalmente no mobile. Sombra suave + border hairline dão profundidade.
//
// Nota: o path da screenshot é resolvido via useBaseUrl para respeitar o
// basePath do Docusaurus (ex: /resma/img/...). Sem isso, a imagem 404 quando
// o site é servido sob um subpath.

import useBaseUrl from '@docusaurus/useBaseUrl';

type BrowserMockupProps = {
  /** URL exibida na chrome bar (sem protocolo). */
  url: string;
  /** Path da screenshot relativo a /static (ex: "/img/dashboard-screenshot.svg"). */
  screenshot: string;
  /** Alt text para a screenshot (a11y). */
  alt?: string;
};

const dotStyles: Record<'red' | 'yellow' | 'green', string> = {
  red: 'bg-[#ff5f56]',
  yellow: 'bg-[#ffbd2e]',
  green: 'bg-[#27c93f]',
};

export default function BrowserMockup({
  url,
  screenshot,
  alt = 'RESMA dashboard preview',
}: BrowserMockupProps): React.JSX.Element {
  const screenshotSrc = useBaseUrl(screenshot);
  return (
    <div className="mx-auto w-full overflow-hidden rounded-xl border border-hairline bg-surface-2 shadow-2xl shadow-black/40">
      {/* Chrome bar */}
      <div className="flex h-10 items-center gap-2 border-b border-hairline bg-surface-3 px-4">
        <span className={`h-3 w-3 rounded-full ${dotStyles.red}`} />
        <span className={`h-3 w-3 rounded-full ${dotStyles.yellow}`} />
        <span className={`h-3 w-3 rounded-full ${dotStyles.green}`} />
        <div className="ml-3 flex h-6 flex-1 items-center rounded-md bg-surface-1 px-3">
          <span
            className="truncate font-mono text-xs text-muted"
            aria-label={`URL: ${url}`}>
            {url}
          </span>
        </div>
        {/* Refresh icon (decorativo) */}
        <svg
          className="h-4 w-4 shrink-0 text-muted"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true">
          <path d="M21 12a9 9 0 1 1-2.64-6.36" />
          <path d="M21 3v6h-6" />
        </svg>
      </div>
      {/* Viewport / screenshot */}
      <div className="bg-surface-1">
        <img
          src={screenshotSrc}
          alt={alt}
          loading="lazy"
          className="block h-auto w-full"
        />
      </div>
    </div>
  );
}
