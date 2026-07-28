// T10 (uiux): Final CTA — seção final de conversão com DottedBackground,
// social proof badges e CTAs "Get Started" + "Star on GitHub".
//
// Reusa o DottedBackground criado em T8 (componente criado lá, usado aqui).
// Reusa os estilos ctaPrimary/ctaSecondary do index.module.css (mesmas pills
// do hero) para consistência visual. Posicionado após Comparison Table (T7).

import Link from '@docusaurus/Link';
import {Star} from 'lucide-react';
import DottedBackground from '../effects/DottedBackground';
import SocialProofBadges from './SocialProofBadges';
import Heading from '@theme/Heading';
import styles from '../../pages/index.module.css';
import clsx from 'clsx';

export default function FinalCTA(): React.JSX.Element {
  return (
    <section className="relative overflow-hidden py-24 lg:py-32">
      {/* T8: dotted glow background sutil atrás do CTA. */}
      <DottedBackground color="#2563eb" opacity={0.1} />
      <div className="container relative z-10 text-center">
        <SocialProofBadges />
        <Heading
          as="h2"
          className="mt-8 text-3xl font-semibold tracking-tight text-ink lg:text-5xl">
          Ready to right-size your Swarm?
        </Heading>
        <p className="mx-auto mt-4 max-w-2xl text-body">
          Deploy RESMA in minutes. MIT-licensed, self-hosted, no external
          dependencies.
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-4">
          <Link
            className={clsx(styles.ctaPrimary, 'button')}
            to="/docs/introduction">
            Get Started
          </Link>
          <a
            className={clsx(styles.ctaSecondary, 'button')}
            href="https://github.com/resma-swarm/resma">
            <Star size={18} className="mr-2 inline" aria-hidden="true" />
            Star on GitHub
          </a>
        </div>
      </div>
    </section>
  );
}
