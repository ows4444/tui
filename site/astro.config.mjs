// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const repo = 'https://github.com/ows4444/tui';
const site = 'https://tui.nizaami.com';
const description =
	'tui is a terminal UI (TUI) framework for Go built on the Elm Architecture: Model, Update and View. Standard library only, with 40+ widgets, flex and grid layout, theming, a cell-diff renderer, accessible output and a headless test harness.';

// Structured data for search engines and AI assistants: the site, and the Go
// library it documents. Every field is a fact from the repository.
const jsonLd = {
	'@context': 'https://schema.org',
	'@graph': [
		{
			'@type': 'WebSite',
			'@id': `${site}/#website`,
			url: `${site}/`,
			name: 'tui',
			alternateName: 'tui: terminal UI framework for Go',
			description,
			inLanguage: 'en',
			about: { '@id': `${site}/#software` },
		},
		{
			'@type': 'SoftwareSourceCode',
			'@id': `${site}/#software`,
			name: 'tui',
			description,
			url: `${site}/`,
			codeRepository: repo,
			programmingLanguage: { '@type': 'ComputerLanguage', name: 'Go', url: 'https://go.dev' },
			runtimePlatform: 'Go 1.26 or later',
			operatingSystem: 'Linux, macOS, Windows, FreeBSD, OpenBSD, NetBSD, DragonFly BSD',
			license: `${repo}/blob/code/LICENSE`,
			keywords: 'Go, Golang, TUI, terminal UI, terminal user interface, CLI, Elm Architecture, widgets, ANSI, accessibility',
			image: `${site}/og.png`,
		},
	],
};

export default defineConfig({
	site,
	integrations: [
		starlight({
			title: 'tui',
			description,
			components: {
				Hero: './src/components/Hero.astro',
				Footer: './src/components/Footer.astro',
			},
			head: [
				{ tag: 'link', attrs: { rel: 'preconnect', href: 'https://fonts.googleapis.com' } },
				{ tag: 'link', attrs: { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: true } },
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=Geist:wght@400..700&family=JetBrains+Mono:wght@400..700&display=swap',
					},
				},
				{ tag: 'meta', attrs: { property: 'og:image', content: `${site}/og.png` } },
				{ tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
				{ tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
				{ tag: 'meta', attrs: { property: 'og:image:alt', content: 'tui, a terminal UI framework for Go, with a dashboard built in it' } },
				{ tag: 'meta', attrs: { name: 'twitter:image', content: `${site}/og.png` } },
				{ tag: 'meta', attrs: { name: 'theme-color', content: '#0b0f14' } },
				{ tag: 'link', attrs: { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' } },
				{ tag: 'link', attrs: { rel: 'alternate', type: 'text/plain', title: 'llms.txt', href: '/llms.txt' } },
				{ tag: 'script', attrs: { type: 'application/ld+json' }, content: JSON.stringify(jsonLd) },
			],
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
						{ label: 'What v1.0 means', link: '/v1/' },
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
