// T2 (uiux): Terminal mockup estático para o hero.
//
// Layout split do hero — lado direito mostra um terminal estilizado com a
// saída de `docker stack deploy -c resma.yml resma`. T9 adiciona animação
// de typewriter (digitação) iniciada quando o terminal entra no viewport.
// T8 adiciona border beam na borda.
//
// Estrutura:
//   header  -> 3 dots (red/yellow/green) + título ~/resma -- zsh
//   body    -> prompt $ em accent green + output em text-body + success em --resma-success

import {useRef} from 'react';
import {useInView} from 'framer-motion';
import BorderBeam from '../effects/BorderBeam';
import Typewriter from '../animations/Typewriter';

type TerminalLine = {
  text: string;
  variant?: 'prompt' | 'output' | 'success';
};

const lines: TerminalLine[] = [
  {text: 'docker stack deploy -c resma.yml resma', variant: 'prompt'},
  {text: 'Creating network resma_default', variant: 'output'},
  {text: 'Creating service resma_api', variant: 'output'},
  {text: 'Creating service resma_ml', variant: 'output'},
  {text: 'Creating service resma_agent', variant: 'output'},
  {text: 'OK RESMA deployed -- dashboard at :8080', variant: 'success'},
];

const dotStyles: Record<'red' | 'yellow' | 'green', string> = {
  red: 'bg-[#ef4444]',
  yellow: 'bg-[#f59e0b]',
  green: 'bg-[#3ecf8e]',
};

function renderLine(line: TerminalLine): React.JSX.Element {
  if (line.variant === 'prompt') {
    return (
      <span className="text-body">
        <span className="text-[#3ecf8e]">$</span>{' '}
        <span>{line.text}</span>
      </span>
    );
  }
  if (line.variant === 'success') {
    return <span className="text-success">{line.text}</span>;
  }
  return <span className="text-body">{line.text}</span>;
}

export default function TerminalMockup(): React.JSX.Element {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, {once: true, margin: '-50px'});

  return (
    <div
      ref={ref}
      className="relative rounded-lg border border-hairline bg-surface-2 shadow-2xl shadow-black/40 overflow-hidden font-mono text-sm">
      {/* T8: border beam — beam de luz viajando na borda. */}
      <BorderBeam />
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-hairline bg-surface-3">
        <span className={`h-3 w-3 rounded-full ${dotStyles.red}`} />
        <span className={`h-3 w-3 rounded-full ${dotStyles.yellow}`} />
        <span className={`h-3 w-3 rounded-full ${dotStyles.green}`} />
        <span className="ml-2 text-xs text-muted">~/resma -- zsh</span>
      </div>
      {/* Body — T9: typewriter iniciado quando in-view. */}
      <div className="px-4 py-4">
        <Typewriter lines={lines} start={inView} renderLine={renderLine} />
      </div>
    </div>
  );
}
