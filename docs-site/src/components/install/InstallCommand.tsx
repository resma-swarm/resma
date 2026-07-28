// T4 (uiux): Seção Install com tabs Swarm/Compose/Standalone e copy button.
//
// Bloco copiável que reduz barreira de entrada: 3 tabs para os métodos de
// deploy (Docker Swarm default, Docker Compose, Standalone dev) e um botão
// de copiar com feedback visual (Copy -> Check por 2s).
//
// Estilo segue a surface ladder do redesign (canvas -> surface-1 -> surface-2
// -> surface-3) com hairline borders. Sem animação de typewriter (T9) e sem
// tracking de evento (T10) — fora de escopo desta task.
//
// Referências:
//   - react-install-command (https://github.com/TimMikeladze/react-install-command)
//   - Coolify install section (https://coolify.io)
//   - shadcn Tabs pattern (https://ui.shadcn.com/docs/components/tabs)
//
// Decisão: implementar tabs custom com shadcn pattern + navigator.clipboard
// (não usar a lib react-install-command para evitar dependência extra).

import {useState} from 'react';
import {Check, Copy} from 'lucide-react';
import clsx from 'clsx';

type InstallTab = {
  id: 'swarm' | 'compose' | 'standalone';
  label: string;
  command: string;
};

const tabs: InstallTab[] = [
  {
    id: 'swarm',
    label: 'Swarm',
    command: 'docker stack deploy -c resma.yml resma',
  },
  {
    id: 'compose',
    label: 'Compose',
    command: 'docker compose up -d',
  },
  {
    id: 'standalone',
    label: 'Standalone',
    command:
      'git clone https://github.com/resma-swarm/resma.git && cd resma && docker compose -f docker-compose.standalone.yml up -d',
  },
];

export default function InstallCommand(): React.JSX.Element {
  const [active, setActive] = useState<InstallTab['id']>('swarm');
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
            One command to deploy RESMA on Docker Swarm, Compose, or standalone.
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
                      'flex-shrink-0 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                      isActive
                        ? 'bg-surface-3 text-ink'
                        : 'text-muted hover:text-body',
                    )}>
                    {tab.label}
                  </button>
                );
              })}
            </div>

            {/* Command + copy */}
            <div
              role="tabpanel"
              id={`install-panel-${activeTab.id}`}
              aria-labelledby={`install-tab-${activeTab.id}`}
              className="flex items-center gap-3 px-4 py-4 font-mono text-sm">
              <span className="flex-shrink-0 text-brand-accent">$</span>
              <code className="flex-1 break-all text-body">
                {activeTab.command}
              </code>
              <button
                type="button"
                onClick={handleCopy}
                aria-label={copied ? 'Copied' : 'Copy command'}
                className={clsx(
                  'flex-shrink-0 flex items-center justify-center h-8 w-8 rounded-md',
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
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
