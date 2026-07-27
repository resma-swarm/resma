import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'introduction',
        'installation',
        'configuration',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'api-reference',
      ],
    },
  ],
};

export default sidebars;
