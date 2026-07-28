import clsx from 'clsx';
import type {ReactNode} from 'react';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';

import AuroraBackground from '../components/hero/AuroraBackground';
import BadgeStack from '../components/hero/BadgeStack';
import TerminalMockup from '../components/hero/TerminalMockup';
import styles from './index.module.css';

type FeatureItem = {
  title: string;
  description: ReactNode;
  icon: string;
};

const features: FeatureItem[] = [
  {
    title: 'Metrics',
    icon: '📊',
    description: (
      <>
        Continuous collection of CPU and memory metrics from Docker Swarm
        services. Time-series storage in DuckDB with configurable retention
        and per-container granularity.
      </>
    ),
  },
  {
    title: 'ML Recommendations',
    icon: '🤖',
    description: (
      <>
        Statistical analysis and machine learning models (scikit-learn) suggest
        optimal resource limits based on historical usage patterns, reducing
        waste and preventing OOM kills.
      </>
    ),
  },
  {
    title: 'Leak Detection',
    icon: '🔍',
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
    icon: '📈',
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
    icon: '🔌',
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
    icon: '🔓',
    description: (
      <>
        MIT-licensed and fully open source. Deploy on Docker Swarm or Docker
        Compose. No external dependencies beyond Docker and the ML sidecar.
      </>
    ),
  },
];

function Feature({title, description, icon}: FeatureItem) {
  return (
    <div className={clsx('col col--4', styles.feature)}>
      <div className={styles.featureCard}>
        <div className={styles.featureIcon}>{icon}</div>
        <Heading as="h3" className={styles.featureTitle}>
          {title}
        </Heading>
        <p className={styles.featureDescription}>{description}</p>
      </div>
    </div>
  );
}

function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {features.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
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
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
