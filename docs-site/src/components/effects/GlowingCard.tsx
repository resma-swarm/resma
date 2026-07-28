// T8 (uiux): Glowing Card — wrapper que aplica um glow radial seguindo o cursor.
//
// Recria o "Glowing Effect" da Aceternity com CSS + JS mínimo (sem dependências).
// No mousemove, atualiza --glow-x/--glow-y; um overlay radial-gradient aparece
// com opacity 0 -> 100 no hover do parent (class `group`). Respeita a paleta
// RESMA (accent verde #3ecf8e em 15% de opacidade — sutil, estilo Linear).
//
// Uso: <GlowingCard className="..."> <Card/> </GlowingCard>. O parent deve ter
// a class `group` (ou o GlowingCard inclui `group` — aqui incluímos para
// garantir o hover mesmo sem `group` no ancestor).

import {useRef} from 'react';
import type {ReactNode} from 'react';

type GlowingCardProps = {
  children: ReactNode;
  className?: string;
};

export default function GlowingCard({
  children,
  className,
}: GlowingCardProps): React.JSX.Element {
  const ref = useRef<HTMLDivElement>(null);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = ref.current?.getBoundingClientRect();
    if (!rect) return;
    ref.current?.style.setProperty('--glow-x', `${e.clientX - rect.left}px`);
    ref.current?.style.setProperty('--glow-y', `${e.clientY - rect.top}px`);
  };

  return (
    <div
      ref={ref}
      onMouseMove={handleMouseMove}
      className={`group relative ${className ?? ''}`}>
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 rounded-xl opacity-0 transition-opacity duration-300 group-hover:opacity-100"
        style={{
          background:
            'radial-gradient(300px circle at var(--glow-x, 50%) var(--glow-y, 50%), rgba(62, 207, 142, 0.15), transparent 40%)',
        }}
      />
      {children}
    </div>
  );
}
