// T8 (uiux): Dotted Background — pattern de pontos radiais com mask radial.
//
// Recria o "Dot Pattern" da Magic UI com CSS puro (radial-gradient + mask).
// Componente criado aqui (T8) e usado em T10 (Final CTA section) como background
// decorativo atrás do CTA. Sutil, não atrapalha legibilidade (mask radial
// esmaece nas bordas).

type DottedBackgroundProps = {
  /** Cor dos pontos (azul brand por padrão). */
  color?: string;
  /** Opacidade do pattern (0-1). */
  opacity?: number;
  /** Espaçamento entre pontos em px. */
  size?: number;
};

export default function DottedBackground({
  color = '#2563eb',
  opacity = 0.15,
  size = 24,
}: DottedBackgroundProps): React.JSX.Element {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0"
      style={{
        backgroundImage: `radial-gradient(circle, ${color} 1px, transparent 1px)`,
        backgroundSize: `${size}px ${size}px`,
        opacity,
        maskImage:
          'radial-gradient(ellipse at center, black 30%, transparent 70%)',
        WebkitMaskImage:
          'radial-gradient(ellipse at center, black 30%, transparent 70%)',
      }}
    />
  );
}
