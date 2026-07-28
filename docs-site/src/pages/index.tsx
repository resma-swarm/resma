import clsx from 'clsx';
import type {ReactNode} from 'react';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';
import {
  Activity,
  BrainCircuit,
  LayoutDashboard,
  SearchCheck,
  Unlock,
  Webhook,
} from 'lucide-react';
import type {LucideIcon} from 'lucide-react';

import AuroraBackground from '../components/hero/AuroraBackground';
import BadgeStack from '../components/hero/BadgeStack';
import TerminalMockup from '../components/hero/TerminalMockup';
import InstallCommand from '../components/install/InstallCommand';
import BrowserMockup from '../components/dashboard/BrowserMockup';
import styles from './index.module.css';

// T3 (uiux): Feature cards — bento grid assimétrico com Lucide icons.
//
// Substitui os 6 cards antigos (emojis + Infima grid) por um bento grid
// Tailwind (1 col mobile / 2 col md / 4 col lg) com hairline borders e
// surface ladder, inspirado em Linear/Komodo. O card "ML Recommendations"
// é o destaque (2 col x 2 row no desktop) e inclui um callout mono com
// exemplo de recomendação.

type FeatureItem = {
  title: string;
  description: ReactNode;
  icon: LucideIcon;
  /** Classes responsivas de col-span/row-span (Tailwind). */
  span: string;
};

type LargeFeatureItem = FeatureItem & {
  callout: ReactNode;
};

const smallFeatures: FeatureItem[] = [
  {
    title: 'Metrics',
    icon: Activity,
    span: 'col-span-1 md:col-span-1 lg:col-span-2',
    description: (
      <>
        Continuous collection of CPU and memory metrics from Docker Swarm
        services. Time-series storage in DuckDB with configurable retention
        and per-container granularity.
      </>
    ),
  },
  {
    title: 'Leak Detection',
    icon: SearchCheck,
    span: 'col-span-1 md:col-span-1 lg:col-span-2',
    description: (
      <>
        Automated detection of memory leak signatures using trend analysis and
        linear regression on memory time-series. Alerts before containers hit
        hard limits.
      </>
    ),
  },
  {
    title: 'Dashboard',
    icon: LayoutDashboard,
    span: 'col-span-1 md:col-span-1 lg:col-span-1',
    description: (
      <>
        Real-time React dashboard with Server-Sent Events (SSE) streaming.
        Visualize resource trends, recommendations, and leak alerts across all
        Swarm services.
      </>
    ),
  },
  {
    title: 'API',
    icon: Webhook,
    span: 'col-span-1 md:col-span-1 lg:col-span-1',
    description: (
      <>
        Go API with JWT auth for the internal UI and API-key + scopes for the
        public <code>/api/v1/*</code> surface. OpenAPI documentation via
        swaggo.
      </>
    ),
  },
  {
    title: 'Open Source',
    icon: Unlock,
    span: 'col-span-1 md:col-span-2 lg:col-span-2',
    description: (
      <>
        MIT-licensed and fully open source. Deploy on Docker Swarm or Docker
        Compose. No external dependencies beyond Docker and the ML sidecar.
      </>
    ),
  },
];

const largeFeature: LargeFeatureItem = {
  title: 'ML Recommendations',
  icon: BrainCircuit,
  span: 'col-span-1 md:col-span-2 lg:col-span-2 row-span-1 md:row-span-2 lg:row-span-2',
  description: (
    <>
      Statistical analysis and machine learning models (scikit-learn) suggest
      optimal resource limits based on historical usage patterns, reducing
      waste and preventing OOM kills.
    </>
  ),
  callout: (
    <>
      Service <span className="text-ink">api</span> should set{' '}
      <span className="text-accent">memory_limit</span> to 512MB
      <br />
      <span className="text-muted">
        current p95: 420MB · trend: +2.1%/week
      </span>
    </>
  ),
};

function FeatureIcon({
  icon: Icon,
  large = false,
}: {
  icon: LucideIcon;
  large?: boolean;
}) {
  return (
    <div
      className={clsx(
        'mb-4 flex h-10 w-10 items-center justify-center rounded-lg',
        large
          ? 'bg-accent/10 text-accent'
          : 'bg-brand/10 text-brand',
      )}>
      <Icon className="h-5 w-5" aria-hidden="true" />
    </div>
  );
}

function FeatureCard({title, description, icon, span}: FeatureItem) {
  return (
    <div
      className={clsx(
        'group flex flex-col rounded-xl border border-hairline bg-surface-2 p-6',
        'transition-[border-color,transform] duration-200 ease-out',
        'hover:border-hairline-strong hover:-translate-y-0.5',
        span,
      )}>
      <FeatureIcon icon={icon} />
      <Heading
        as="h3"
        className="mb-2 text-lg font-semibold tracking-tight text-ink">
        {title}
      </Heading>
      <p className="text-sm leading-relaxed text-body">{description}</p>
    </div>
  );
}

function FeatureCardLarge({title, description, icon, span, callout}: LargeFeatureItem) {
  return (
    <div
      className={clsx(
        'group flex flex-col rounded-xl border border-hairline p-6',
        styles.featureCardLarge,
        'transition-[border-color,transform] duration-200 ease-out',
        'hover:border-hairline-strong hover:-translate-y-0.5',
        span,
      )}>
      <FeatureIcon icon={icon} large />
      <Heading
        as="h3"
        className="mb-2 text-xl font-semibold tracking-tight text-ink">
        {title}
      </Heading>
      <p className="text-sm leading-relaxed text-body">{description}</p>
      <div className="mt-4 rounded border border-hairline bg-surface-1 p-3 font-mono text-xs leading-relaxed text-body">
        {callout}
      </div>
    </div>
  );
}

function HomepageFeatures() {
  return (
    <section className="py-20 lg:py-28">
      <div className="container">
        <Heading
          as="h2"
          className="mb-12 text-center text-3xl font-semibold tracking-tight text-ink lg:text-4xl">
          Everything you need to manage Swarm resources
        </Heading>
        <div className="grid grid-cols-1 gap-4 auto-rows-fr md:grid-cols-2 lg:grid-cols-4">
          <FeatureCardLarge {...largeFeature} />
          {smallFeatures.map((props) => (
            <FeatureCard key={props.title} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}

// T5 (uiux): Dashboard browser mockup section.
//
// Mostra o produto em ação dentro de um browser frame custom. Screenshot
// estático (SVG) do dashboard RESMA — sidebar, KPIs, time-series chart,
// ML recommendations panel e services table. Posicionado após Features.
function HomepageDashboard() {
  return (
    <section className="py-20 lg:py-28">
      <div className="container">
        <div className="mx-auto max-w-4xl">
          <Heading
            as="h2"
            className="mb-4 text-center text-3xl font-semibold tracking-tight text-ink lg:text-4xl">
            Real-time dashboard
          </Heading>
          <p className="mx-auto mb-12 max-w-2xl text-center text-body">
            Visualize resource trends, ML recommendations, and leak alerts
            across all Swarm services. Updates in real-time via SSE.
          </p>
          <BrowserMockup
            url="resma.local:8080/dashboard"
            screenshot="/img/dashboard-screenshot.svg"
            alt="RESMA dashboard showing CPU and memory time-series, ML recommendations, and services table"
          />
        </div>
      </div>
    </section>
  );
}

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx(styles.heroSplit, 'hero')}>
      <AuroraBackground />
      <div className={clsx('container', styles.heroContainer)}>
        <div className={styles.heroGrid}>
          {/* Coluna esquerda: badges + H1 + subheadline + CTAs */}
          <div className={styles.heroLeft}>
            <BadgeStack />
            <Heading as="h1" className={styles.heroTitle}>
              {siteConfig.title}
            </Heading>
            <p className={styles.heroSubtitle}>{siteConfig.tagline}</p>
            <p className={styles.heroTagline}>
              Metrics, ML-driven resource recommendations, and memory leak
              detection for Docker Swarm — open source and self-hosted.
            </p>
            <div className={styles.heroButtons}>
              <Link
                className={clsx(styles.ctaPrimary, 'button')}
                to="/docs/introduction">
                Get Started
              </Link>
              <Link
                className={clsx(styles.ctaSecondary, 'button')}
                href="https://github.com/resma-swarm/resma">
                Star on GitHub
              </Link>
            </div>
          </div>
          {/* Coluna direita: terminal mockup (oculto no mobile) */}
          <div className={styles.heroRight}>
            <TerminalMockup />
          </div>
        </div>
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} — ${siteConfig.tagline}`}
      description="RESource MAnager for Docker Swarm — metrics, ML recommendations, and memory leak detection.">
      <HomepageHeader />
      <main>
        <InstallCommand />
        <HomepageFeatures />
        <HomepageDashboard />
      </main>
    </Layout>
  );
}
