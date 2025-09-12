// @ts-check

import { themes as prismThemes } from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  // ...existing config...
  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */ ({
        docs: {
          sidebarPath: require.resolve('./sidebars.ts'),
          routeBasePath: 'docs',
          editUrl:
            'https://github.com/dovidio/colino/tree/main/website/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
        // Enable local search
        // See: https://github.com/easyops-cn/docusaurus-search-local
        // npm install --save docusaurus-search-local
        plugins: [
          [
            require.resolve('docusaurus-search-local'),
            /** @type {import('docusaurus-search-local').PluginOptions} */ ({
              hashed: true,
              indexDocs: true,
              indexPages: true,
              docsRouteBasePath: 'docs',
            }),
          ],
        ],
      }),
    ],
  ],
  // ...existing config...
};

export default config;
