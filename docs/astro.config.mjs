import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://dev.standardbeagle.com',
  base: '/claude-go',
  integrations: [
    starlight({
      title: 'Claude Go SDK',
      description: 'A Go SDK for automating Claude Code CLI with concurrent session management, real-time streaming, custom tools, and hooks.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/standardbeagle/claude-go' },
      ],
      // Per-page og:title / og:description / og:url live in each page's own
      // frontmatter `head` (see src/content/docs/*.mdx) so social previews
      // reflect the page being shared, not always the homepage. Only the
      // tags that are genuinely the same on every page belong here.
      head: [
        {
          tag: 'meta',
          attrs: { property: 'og:type', content: 'website' },
        },
        {
          tag: 'meta',
          attrs: { name: 'twitter:card', content: 'summary' },
        },
      ],
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'Start Here',
          items: [
            { label: 'Getting Started', slug: 'getting-started' },
            { label: 'Core Concepts', slug: 'core-concepts' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Examples', slug: 'examples' },
            { label: 'API Overview', slug: 'api-reference' },
          ],
        },
      ],
    }),
  ],
});
