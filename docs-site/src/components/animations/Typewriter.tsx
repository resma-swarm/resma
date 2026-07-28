// T9 (uiux): Typewriter — animação de digitação para o terminal mockup do hero.
//
// Digita as linhas do terminal uma a uma, caractere por caractere, com cursor
// pulsante. Só inicia quando o terminal está visível (prop `start` controlada
// pelo parent via useInView do Framer Motion).
//
// prefers-reduced-motion: se o usuário preferir movimento reduzido, mostra
// todas as linhas imediatamente (sem animação).
//
// Renderização por linha delegada ao parent via `renderLine` para preservar
// o styling (prompt $ em accent, success em --resma-success, etc).

import {useEffect, useState} from 'react';

type LineVariant = 'prompt' | 'output' | 'success';

type TypewriterLine = {
  text: string;
  variant: LineVariant;
};

type TypewriterProps = {
  lines: TypewriterLine[];
  /** Velocidade de digitação em ms por caractere. */
  speed?: number;
  /** Inicia a animação apenas quando true (controlado pelo parent). */
  start: boolean;
  /** Renderiza uma linha com styling por variant. */
  renderLine: (line: TypewriterLine) => React.JSX.Element;
};

export default function Typewriter({
  lines,
  speed = 30,
  start,
  renderLine,
}: TypewriterProps): React.JSX.Element {
  const [displayedCount, setDisplayedCount] = useState(0);
  const [currentText, setCurrentText] = useState('');
  const [lineIdx, setLineIdx] = useState(0);
  const [charIdx, setCharIdx] = useState(0);
  const [reducedMotion, setReducedMotion] = useState(false);

  // Detecta prefers-reduced-motion uma vez.
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    setReducedMotion(mq.matches);
    const handler = (e: MediaQueryListEvent) => setReducedMotion(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // Reduced motion: mostra tudo imediatamente.
  useEffect(() => {
    if (reducedMotion) {
      setDisplayedCount(lines.length);
      setCurrentText('');
      setLineIdx(lines.length);
      setCharIdx(0);
    }
  }, [reducedMotion, lines]);

  // Animação de digitação — só inicia quando `start` é true.
  useEffect(() => {
    if (!start || reducedMotion) return;
    if (lineIdx >= lines.length) return;
    const line = lines[lineIdx];
    if (charIdx < line.text.length) {
      const timer = setTimeout(() => {
        setCurrentText(line.text.slice(0, charIdx + 1));
        setCharIdx(charIdx + 1);
      }, speed);
      return () => clearTimeout(timer);
    }
    // Linha completa — avança para a próxima.
    setDisplayedCount((c) => c + 1);
    setCurrentText('');
    setLineIdx((i) => i + 1);
    setCharIdx(0);
  }, [charIdx, lineIdx, lines, speed, start, reducedMotion]);

  return (
    <div className="space-y-1.5">
      {lines.slice(0, displayedCount).map((line, i) => (
        <div key={i}>{renderLine(line)}</div>
      ))}
      {lineIdx < lines.length && !reducedMotion && start && (
        <div>
          {renderLine({text: currentText, variant: lines[lineIdx].variant})}
          <span className="animate-pulse text-accent">|</span>
        </div>
      )}
    </div>
  );
}
