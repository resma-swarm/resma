import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'RESMA',
  tagline: 'RESource MAnager for Docker Swarm',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://resma-swarm.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  baseUrl: '/resma/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'resma-swarm', // Usually your GitHub org/user name.
  projectName: 'resma', // Usually your repo name.

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // T1 (uiux): plugin PostCSS do Tailwind CSS v4.
  // Empurra @tailwindcss/postcss para a pipeline do Docusaurus sem remover os
  // plugins nativos.
  plugins: ['./src/plugins/tailwind-config.js'],

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/resma-swarm/resma/tree/main/docs-site/',
        },
        blog: false,
        theme: {
          customCss: ['./src/css/custom.css', './src/css/tailwind.css'],
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    // T10 (uiux): metadados SEO — OG + Twitter cards.
    // URL de produção alinhada com o repo github.com/resma-swarm/resma.
    metadata: [
      {name: 'og:title', content: 'RESMA — RESource MAnager for Docker Swarm'},
      {
        name: 'og:description',
        content:
          'Metrics, ML recommendations, and memory leak detection for Docker Swarm. Open-source, self-hosted.',
      },
      {
        name: 'og:image',
        content: 'https://resma-swarm.github.io/resma/img/og-image.png',
      },
      {name: 'og:type', content: 'website'},
      {name: 'og:url', content: 'https://resma-swarm.github.io/resma/'},
      {name: 'twitter:card', content: 'summary_large_image'},
      {
        name: 'twitter:title',
        content: 'RESMA — RESource MAnager for Docker Swarm',
      },
      {
        name: 'twitter:description',
        content:
          'Metrics, ML recommendations, and memory leak detection for Docker Swarm.',
      },
      {
        name: 'twitter:image',
        content: 'https://resma-swarm.github.io/resma/img/og-image.png',
      },
    ],
    navbar: {
      title: 'RESMA',
      logo: {
        alt: 'RESMA Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          to: '/',
          position: 'left',
          label: 'Home',
        },
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Docs',
        },
        {
          to: '/docs/api-reference',
          position: 'left',
          label: 'API',
        },
        {
          href: 'https://github.com/resma-swarm/resma',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Introduction',
              to: '/docs/introduction',
            },
            {
              label: 'Installation',
              to: '/docs/installation',
            },
            {
              label: 'API Reference',
              to: '/docs/api-reference',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/resma-swarm/resma',
            },
            {
              label: 'Issues',
              href: 'https://github.com/resma-swarm/resma/issues',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/resma-swarm/resma',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} RESMA. Built with Docusaurus. Released under the MIT License.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['go', 'python', 'yaml', 'bash', 'docker'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
