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
      head: [
        {
          tag: 'meta',
          attrs: { property: 'og:title', content: 'Claude Go SDK' },
        },
        {
          tag: 'meta',
          attrs: { property: 'og:description', content: 'A Go SDK for automating Claude Code CLI with concurrent session management, real-time streaming, custom tools, and hooks.' },
        },
        {
          tag: 'meta',
          attrs: { property: 'og:type', content: 'website' },
        },
        {
          tag: 'meta',
          attrs: { property: 'og:url', content: 'https://dev.standardbeagle.com/claude-go/' },
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
