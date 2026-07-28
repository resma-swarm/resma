// T8 (uiux): Border Beam — beam de luz viajando na borda de um container.
//
// Recria o efeito "Border Beam" da Magic UI com CSS puro (conic-gradient +
// mask-composite + @property --beam-angle animada). Sem dependências extras.
//
// Uso: envolver o container alvo com `relative` e renderizar <BorderBeam />
// dentro. O beam percorre a borda em `duration` segundos. Respeita
// prefers-reduced-motion (keyframe desabilitado em custom.css).
//
// Fallback: se @property --beam-angle não for suportado (Safari < 16.4),
// o beam aparece estático (sem rotação) — ainda renderiza um arco colorido
// na borda, apenas sem animação. Aceitável como graceful degradation.

type BorderBeamProps = {
  /** Duração da rotação em segundos. */
  duration?: number;
  /** Cor inicial do beam (azul brand por padrão). */
  colorFrom?: string;
  /** Cor final do beam (verde accent por padrão). */
  colorTo?: string;
};

export default function BorderBeam({
  duration = 8,
  colorFrom = '#2563eb',
  colorTo = '#3ecf8e',
}: BorderBeamProps): React.JSX.Element {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 rounded-lg overflow-hidden"
      style={{
        background: `conic-gradient(from var(--beam-angle, 0deg), transparent 0%, ${colorFrom} 25%, ${colorTo} 50%, transparent 75%)`,
        animation: `beam-rotate ${duration}s linear infinite`,
        // Máscara: pinta apenas a borda de 1px (content-box + exclude).
        WebkitMask:
          'linear-gradient(black, black) content-box, linear-gradient(black, black)',
        WebkitMaskComposite: 'xor',
        mask: 'linear-gradient(black, black) content-box, linear-gradient(black, black)',
        maskComposite: 'exclude',
        padding: '1px',
      }}
    />
  );
}
