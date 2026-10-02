// Generates the site's pages from the repository, so the Markdown in the repo
// stays the only copy of the docs:
//
//   - README.md, CONTRIBUTING.md, CHANGELOG.md and docs/**/*.md become pages
//     under src/content/docs/ (the first "# Heading" becomes the page title;
//     relative links are rewritten to site routes or to GitHub).
//   - examples/*/testdata/size-80x24.golden (screens the example tests check)
//     become the /examples/ gallery and src/generated/screens.json.
//
// Run by `npm run dev` and `npm run build`. Its output is gitignored.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const siteDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = path.resolve(siteDir, '..');
const outDir = path.join(siteDir, 'src/content/docs');
const genDir = path.join(siteDir, 'src/generated');

const GITHUB = 'https://github.com/ows4444/tui';
const BRANCH = 'code';
const SCREEN = 'testdata/size-80x24.golden';

// Repository file -> site page. Every link to one of these files is rewritten
// to its route; links to any other repository path go to GitHub.
const pages = [
	{ src: 'README.md', slug: 'overview', title: 'Overview' },
	{ src: 'docs/cookbook.md', slug: 'cookbook' },
	{ src: 'docs/program.md', slug: 'program' },
	{ src: 'docs/layout.md', slug: 'layout' },
	{ src: 'docs/widgets.md', slug: 'widgets' },
	{ src: 'docs/input.md', slug: 'input' },
	{ src: 'docs/theming.md', slug: 'theming' },
	{ src: 'docs/rendering.md', slug: 'rendering' },
	{ src: 'docs/capabilities.md', slug: 'capabilities' },
	{ src: 'docs/accessibility.md', slug: 'accessibility' },
	{ src: 'docs/migrating-to-v1.md', slug: 'migrating-to-v1' },
	{ src: 'docs/testing.md', slug: 'testing' },
	{ src: 'docs/architecture/overview.md', slug: 'architecture' },
	{ src: 'CONTRIBUTING.md', slug: 'contributing' },
	{ src: 'CHANGELOG.md', slug: 'changelog' },
];

const routes = new Map(pages.map((p) => [p.src, `/${p.slug}/`]));
// docs/README.md only routes readers to the other pages; the sidebar does that here.
routes.set('docs/README.md', '/overview/');
routes.set('docs', '/overview/');

let broken = 0;

function rewriteTarget(target, srcFile) {
	if (/^([a-z][a-z0-9+.-]*:|#|\/)/i.test(target)) return target;
	const hashAt = target.indexOf('#');
	const rel = hashAt < 0 ? target : target.slice(0, hashAt);
	const hash = hashAt < 0 ? '' : target.slice(hashAt);
	const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(srcFile), rel)).replace(/\/$/, '');
	if (routes.has(resolved)) return routes.get(resolved) + hash;
	const abs = path.join(repoDir, resolved);
	if (!fs.existsSync(abs)) {
		console.warn(`sync-docs: ${srcFile}: broken link ${target}`);
		broken++;
		return target;
	}
	const kind = fs.statSync(abs).isDirectory() ? 'tree' : 'blob';
	return `${GITHUB}/${kind}/${BRANCH}/${resolved}${hash}`;
}

// rewriteLinks rewrites inline and reference-style links outside fenced code.
function rewriteLinks(md, srcFile) {
	const parts = md.split(/^(```[\s\S]*?^```)/m);
	return parts
		.map((part, i) => {
			if (i % 2 === 1) return part; // a fenced block
			return part
				.replace(/\]\(([^)\s]+)((?:\s+"[^"]*")?)\)/g, (_, t, title) => `](${rewriteTarget(t, srcFile)}${title})`)
				.replace(/^(\[[^\]]+\]:\s+)(\S+)/gm, (_, label, t) => label + rewriteTarget(t, srcFile));
		})
		.join('');
}

function plainText(md) {
	return md
		.replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
		.replace(/[`*_]/g, '')
		.replace(/\s+/g, ' ')
		.trim();
}

function describe(body) {
	const para = body
		.split(/\n\s*\n/)
		.map((s) => s.trim())
		.find((s) => s && !/^(#|```|\||[-*] |>|<)/.test(s));
	if (!para) return undefined;
	const text = plainText(para);
	const sentence = text.match(/^.+?[.!?](\s|$)/)?.[0].trim() ?? text;
	return sentence.length > 200 ? sentence.slice(0, 197) + '...' : sentence;
}

function frontmatter(fields) {
	const lines = Object.entries(fields)
		.filter(([, v]) => v !== undefined)
		.map(([k, v]) => `${k}: ${JSON.stringify(v)}`);
	return `---\n${lines.join('\n')}\n---\n\n`;
}

function syncPage({ src, slug, title }) {
	const md = fs.readFileSync(path.join(repoDir, src), 'utf8');
	const h1 = md.match(/^# (.+)\n/m);
	const body = h1 ? md.slice(h1.index + h1[0].length) : md;
	const out =
		frontmatter({
			title: title ?? h1?.[1].trim() ?? slug,
			description: describe(body),
			editUrl: `${GITHUB}/edit/${BRANCH}/${src}`,
		}) + rewriteLinks(body.replace(/^\n+/, ''), src);
	fs.mkdirSync(path.dirname(path.join(outDir, `${slug}.md`)), { recursive: true });
	fs.writeFileSync(path.join(outDir, `${slug}.md`), out);
}

// packageDoc returns the package comment of examples/<name>/main.go as text.
function packageDoc(name) {
	const file = path.join(repoDir, 'examples', name, 'main.go');
	if (!fs.existsSync(file)) return '';
	const lines = [];
	for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
		if (line.startsWith('package ')) break;
		if (line.startsWith('//go:build')) continue;
		if (line.startsWith('//')) lines.push(line.replace(/^\/\/ ?/, ''));
		else if (line.trim() === '' && lines.length) lines.push('');
	}
	return lines.join('\n').trim();
}

// firstSentence returns the first sentence of an example's package comment,
// with Go's "Command <name>" opener shown as the name in code; '' if none.
function firstSentence(text) {
	const flat = text.split(/\n\s*\n/)[0].replace(/\s+/g, ' ').replace(/^Command (\S+) /, '`$1` ');
	return flat.match(/^.+?[.!?](\s|$)/)?.[0].trim() ?? flat;
}

function syncExamples() {
	const names = fs
		.readdirSync(path.join(repoDir, 'examples'), { withFileTypes: true })
		.filter((d) => d.isDirectory())
		.map((d) => d.name)
		.sort();
	const screens = {};
	const withScreen = [];
	const without = [];
	for (const name of names) {
		const golden = path.join(repoDir, 'examples', name, SCREEN);
		const desc = firstSentence(packageDoc(name));
		if (fs.existsSync(golden)) {
			const text = fs.readFileSync(golden, 'utf8').replace(/\s+$/, '');
			screens[name] = { text, description: desc };
			withScreen.push(name);
		} else {
			without.push(name);
		}
	}

	let md = frontmatter({
		title: 'Examples',
		description: 'Every program under examples/, with the 80×24 screen its test checks.',
		editUrl: false,
	});
	md += `Every directory under [examples/](${GITHUB}/tree/${BRANCH}/examples) is a runnable program. `;
	md += `Clone the repository and run one from its root, for example \`go run ./examples/dashboard\`.\n\n`;
	md += `The screens below are not mock-ups: each is the example's \`${SCREEN}\` golden file, `;
	md += `the first frame at 80×24 that its own test compares against, so they show text and layout without colour.\n\n`;
	for (const name of withScreen) {
		const { text, description } = screens[name];
		md += `## ${name}\n\n` + (description ? `${description}\n\n` : '');
		md += '````text frame="terminal" title="go run ./examples/' + name + '"\n' + text + '\n````\n\n';
		md += `[Source](${GITHUB}/tree/${BRANCH}/examples/${name})\n\n`;
	}
	if (without.length) {
		md += `## Without a screen capture\n\n`;
		for (const name of without) {
			const desc = firstSentence(packageDoc(name));
			md += `- [${name}](${GITHUB}/tree/${BRANCH}/examples/${name})` + (desc ? `: ${desc}` : '') + '\n';
		}
	}
	fs.writeFileSync(path.join(outDir, 'examples.md'), md);
	fs.mkdirSync(genDir, { recursive: true });
	fs.writeFileSync(path.join(genDir, 'screens.json'), JSON.stringify(screens, null, '\t') + '\n');
	return withScreen.length;
}

// Clear previously generated pages, keeping the hand-written landing page.
for (const entry of fs.readdirSync(outDir)) {
	if (entry !== 'index.mdx') fs.rmSync(path.join(outDir, entry), { recursive: true });
}
for (const page of pages) syncPage(page);
const n = syncExamples();
console.log(`sync-docs: ${pages.length} pages, ${n} example screens`);
if (broken) {
	console.error(`sync-docs: ${broken} broken link(s)`);
	process.exit(1);
}
