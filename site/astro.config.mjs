// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const repo = 'https://github.com/ows4444/tui';

export default defineConfig({
	site: 'https://tui.nizaami.com',
	integrations: [
		starlight({
			title: 'tui',
			description:
				'A terminal UI framework for Go built on the Elm Architecture. Standard library only.',
			logo: { src: './src/assets/logo.svg', alt: 'tui' },
			favicon: '/favicon.svg',
			social: [{ icon: 'github', label: 'GitHub', href: repo }],
			customCss: ['./src/styles/custom.css'],
			// Pages are generated from the repository's Markdown by scripts/sync-docs.mjs;
			// each sets its own editUrl pointing at its source file.
			lastUpdated: false,
			expressiveCode: {
				themes: ['github-dark', 'github-light'],
			},
			sidebar: [
				{
					label: 'Start here',
					items: [
						{ label: 'Overview', link: '/overview/' },
						{ label: 'Examples', link: '/examples/' },
						{ label: 'Cookbook', link: '/cookbook/' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Running a Program', link: '/program/' },
						{ label: 'Layout', link: '/layout/' },
						{ label: 'Widgets', link: '/widgets/' },
						{ label: 'Input and key bindings', link: '/input/' },
						{ label: 'Theming', link: '/theming/' },
						{ label: 'Rendering', link: '/rendering/' },
						{ label: 'Terminal capabilities', link: '/capabilities/' },
						{ label: 'Accessibility', link: '/accessibility/' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'API reference (pkg.go.dev)', link: 'https://pkg.go.dev/github.com/ows4444/tui', attrs: { target: '_blank' } },
						{ label: 'Migrating to v1', link: '/migrating-to-v1/' },
						{ label: 'Changelog', link: '/changelog/' },
					],
				},
				{
					label: 'Contributing',
					items: [
						{ label: 'Contributing', link: '/contributing/' },
						{ label: 'Architecture', link: '/architecture/' },
						{ label: 'Testing', link: '/testing/' },
					],
				},
			],
		}),
	],
});
