import clsx from 'clsx';
import type {ReactNode} from 'react';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';

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
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/introduction">
            Get Started 🚀
          </Link>
          <Link
            className="button button--outline button--lg"
            href="https://github.com/resma-swarm/resma">
            GitHub ⭐
          </Link>
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
