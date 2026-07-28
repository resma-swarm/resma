// T2 (uiux): Aurora background CSS puro para o hero da landing page.
//
// Renderiza duas "bolhas" desfocadas (brand + brand-accent) atrás do conteúdo
// do hero, evocando o estilo Linear/Supabase sem JS pesado. A opacidade é
// intencionalmente baixa (0.15) para manter o visual "premium" e evitar o
// efeito "cheap neon". Respeita prefers-reduced-motion.
//
// Sem animação JS nesta task — T8/T9 podem adicionar movimento sutil.

import type {CSSProperties} from 'react';

const styles: Record<string, CSSProperties> = {
  auroraBg: {
    position: 'absolute',
    inset: 0,
    overflow: 'hidden',
    zIndex: 0,
    pointerEvents: 'none',
  },
  blob: {
    content: "''",
    position: 'absolute',
    width: '600px',
    height: '600px',
    borderRadius: '50%',
    filter: 'blur(120px)',
    opacity: 0.15,
  },
  blobBrand: {
    background: 'var(--resma-brand)',
    top: '-200px',
    left: '-100px',
  },
  blobAccent: {
    background: 'var(--resma-brand-accent)',
    bottom: '-200px',
    right: '-100px',
  },
};

export default function AuroraBackground(): React.JSX.Element {
  return (
    <div className="aurora-bg" style={styles.auroraBg} aria-hidden="true">
      <span style={{...styles.blob, ...styles.blobBrand}} />
      <span style={{...styles.blob, ...styles.blobAccent}} />
    </div>
  );
}
