// T9 (uiux): Reveal — wrapper de scroll-triggered fade+slide com Framer Motion.
//
// Usa LazyMotion + domAnimation (bundle ~15KB vs 34KB do motion completo).
// Anima opacity 0->1 e y 24px->0 quando o elemento entra no viewport
// (whileInView), uma única vez (once: true, margin: '-100px').
//
// prefers-reduced-motion: respeitado pelo Framer Motion automaticamente
// (ele reduz/neutraliza animações quando a media query é reduce). Além disso,
// se o usuário tiver reduced-motion, o whileInView ainda seta opacity:1 ao
// entrar no viewport (apenas sem a transição visual suave).
//
// Uso: <Reveal delay={0.1}><Section/></Reveal>. Hero NÃO recebe Reveal.

import {LazyMotion, domAnimation, motion} from 'framer-motion';
import type {ReactNode} from 'react';

type RevealProps = {
  children: ReactNode;
  /** Delay em segundos (stagger sutil entre seções). */
  delay?: number;
  /** Offset vertical inicial em px. */
  y?: number;
  /** Classe extra repassada ao motion.div (ex: className da seção). */
  className?: string;
};

export default function Reveal({
  children,
  delay = 0,
  y = 24,
  className,
}: RevealProps): React.JSX.Element {
  return (
    <LazyMotion features={domAnimation} strict>
      <motion.div
        className={className}
        initial={{opacity: 0, y}}
        whileInView={{opacity: 1, y: 0}}
        viewport={{once: true, margin: '-100px'}}
        transition={{duration: 0.6, delay, ease: 'easeOut'}}>
        {children}
      </motion.div>
    </LazyMotion>
  );
}
