// T4 (uiux): Seção Install com tabs Installer/Swarm/Compose/Standalone e copy button.
//
// Bloco copiável que reduz barreira de entrada. O método recomendado é o
// installer container (docker run resmaswarm/resma-install:latest), que
// aparece como primeira tab. Os demais métodos (Swarm, Compose, Standalone)
// ficam como alternativas.
//
// Syntax highlighting via prism-react-renderer (já é dependência do
// Docusaurus) com tema dracula — mesmo darkTheme configurado em
// docusaurus.config.ts. A linguagem por tab é declarada em `language` e
// o Prism tokeniza o comando; o copy button copia o texto bruto.
//
// Estilo segue a surface ladder do redesign (canvas -> surface-1 -> surface-2
// -> surface-3) com hairline borders. Sem animação de typewriter (T9) e sem
// tracking de evento (T10) — fora de escopo desta task.
//
// Referências:
//   - prism-react-renderer (https://github.com/FormidableLabs/prism-react-renderer)
//   - Coolify install section (https://coolify.io)
//   - shadcn Tabs pattern (https://ui.shadcn.com/docs/components/tabs)

import {useState} from 'react';
import {Highlight, themes} from 'prism-react-renderer';
import {Check, Copy} from 'lucide-react';
import clsx from 'clsx';

type InstallTab = {
  id: 'installer' | 'standalone' | 'upgrade' | 'uninstall';
  label: string;
  /** Linguagem Prism para highlight. */
  language: 'bash';
  /** Comando bruto copiado para o clipboard (sem $ prefix). */
  command: string;
};

const tabs: InstallTab[] = [
  {
    id: 'installer',
    label: 'Installer',
    language: 'bash',
    command: `docker run -it --rm \\
  --name resma-installer \\
  --volume /var/run/docker.sock:/var/run/docker.sock \\
  resmaswarm/resma-install:latest`,
  },
  {
    id: 'standalone',
    label: 'Standalone',
    language: 'bash',
    command: `# 1. Clone o repositório
git clone https://github.com/resma-swarm/resma.git
cd resma

# 2. Suba o stack standalone (sem workers)
docker compose -f docker-compose.standalone.yml up -d

# 3. Acesse o dashboard em http://localhost:8080`,
  },
  {
    id: 'upgrade',
    label: 'Upgrade',
    language: 'bash',
    command: `docker run -it --rm \\
  --name resma-upgrader \\
  --volume /var/run/docker.sock:/var/run/docker.sock \\
  -e MODE=upgrade \\
  resmaswarm/resma-install:latest`,
  },
  {
    id: 'uninstall',
    label: 'Uninstall',
    language: 'bash',
    command: `docker run -it --rm \\
  --name resma-uninstaller \\
  --volume /var/run/docker.sock:/var/run/docker.sock \\
  -e MODE=uninstall \\
  resmaswarm/resma-install:latest`,
  },
];

export default function InstallCommand(): React.JSX.Element {
  const [active, setActive] = useState<InstallTab['id']>('installer');
  const [copied, setCopied] = useState(false);

  const activeTab = tabs.find((t) => t.id === active) ?? tabs[0];

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(activeTab.command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard pode falhar em contextos sem permissão; silently ignore
    }
  };

  return (
    <section className="py-16 lg:py-20">
      <div className="container">
        <div className="mx-auto max-w-3xl">
          <h2 className="mb-3 text-center text-2xl font-semibold tracking-tight text-ink lg:text-3xl">
            Install
          </h2>
          <p className="mb-8 text-center text-sm text-muted">
            Install, upgrade, or uninstall RESMA on Docker Swarm — installer
            container (recommended), standalone dev setup, in-place upgrade, or
            clean uninstall.
          </p>

          <div className="rounded-lg border border-hairline bg-surface-2 overflow-hidden">
            {/* Tabs */}
            <div
              role="tablist"
              aria-label="Install method"
              className="flex gap-1 p-1 bg-surface-1 border-b border-hairline overflow-x-auto">
              {tabs.map((tab) => {
                const isActive = tab.id === active;
                return (
                  <button
                    key={tab.id}
                    role="tab"
                    type="button"
                    aria-selected={isActive}
                    aria-controls={`install-panel-${tab.id}`}
                    id={`install-tab-${tab.id}`}
                    onClick={() => setActive(tab.id)}
                    className={clsx(
                      'flex-shrink-0 flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                      isActive
                        ? 'bg-surface-3 text-ink'
                        : 'text-muted hover:text-body',
                    )}>
                    {tab.label}
                  </button>
                );
              })}
            </div>

            {/* Command + copy com syntax highlight */}
            <div
              role="tabpanel"
              id={`install-panel-${activeTab.id}`}
              aria-labelledby={`install-tab-${activeTab.id}`}
              className="relative">
              <button
                type="button"
                onClick={handleCopy}
                aria-label={copied ? 'Copied' : 'Copy command'}
                className={clsx(
                  'absolute top-3 right-3 z-10 flex items-center justify-center h-8 w-8 rounded-md',
                  'border border-hairline bg-surface-3 text-muted',
                  'hover:text-ink hover:border-hairline-strong transition-colors',
                  copied && 'text-success border-success/40',
                )}>
                {copied ? (
                  <Check className="h-4 w-4" aria-hidden="true" />
                ) : (
                  <Copy className="h-4 w-4" aria-hidden="true" />
                )}
              </button>
              <Highlight
                theme={themes.dracula}
                code={activeTab.command}
                language={activeTab.language}>
                {({className, style, tokens, getLineProps, getTokenProps}) => (
                  <pre
                    className={clsx(
                      className,
                      'px-4 py-4 pr-12 overflow-x-auto font-mono text-sm leading-relaxed',
                      // override fundo do tema dracula para surface-2 do redesign
                      '!bg-transparent',
                    )}
                    style={{...style, background: 'transparent'}}>
                    {tokens.map((line, i) => {
                      const lineProps = getLineProps({line});
                      return (
                        <div
                          key={i}
                          {...lineProps}
                          className={clsx(lineProps.className, 'table-row')}>
                          <span
                            aria-hidden="true"
                            className="table-cell select-none pr-4 text-right text-muted/60">
                            {i + 1}
                          </span>
                          <span className="table-cell">
                            {line.map((token, key) => {
                              const tokenProps = getTokenProps({token});
                              return <span key={key} {...tokenProps} />;
                            })}
                          </span>
                        </div>
                      );
                    })}
                  </pre>
                )}
              </Highlight>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
