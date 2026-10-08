// Generates the site's pages from the repository, so the Markdown in the repo
// stays the only copy of the docs:
//
//   - README.md, CONTRIBUTING.md, CHANGELOG.md and docs/**/*.md become pages
//     under src/content/docs/ (the first "# Heading" becomes the page title;
//     relative links are rewritten to site routes or to GitHub).
//   - examples/*/testdata/size-80x24.golden (screens the example tests check)
//     become the /examples/ gallery and src/generated/screens.json.
//   - For search engines and AI assistants: each page gets a descriptive
//     <title>, TechArticle JSON-LD and a plain Markdown copy at /<slug>.md;
//     public/llms.txt indexes those copies and public/llms-full.txt joins them
//     (https://llmstxt.org).
//
// Run by `npm run dev` and `npm run build`. Its output is gitignored.

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const siteDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = path.resolve(siteDir, '..');
const outDir = path.join(siteDir, 'src/content/docs');
const genDir = path.join(siteDir, 'src/generated');
const publicDir = path.join(siteDir, 'public');

const SITE = 'https://tui.nizaami.com';
const GITHUB = 'https://github.com/ows4444/tui';
const BRANCH = 'code';
const SCREEN = 'testdata/size-80x24.golden';
const TITLE_SUFFIX = 'tui, the Go terminal UI framework';

// Repository file -> site page. Every link to one of these files is rewritten
// to its route; links to any other repository path go to GitHub. seoTitle
// replaces the default "<title> | tui, the Go terminal UI framework" <title>;
// optional pages go under "Optional" in llms.txt.
const pages = [
	{ src: "README.md", slug: "overview", title: "Overview", seoTitle: 'Install tui and write your first Go terminal app',
		description: "Install tui with go get (Go 1.26+), write a first Model with Init, Update and View, and see every package, its stability level and the supported platforms." },
	{ src: 'docs/cookbook.md', slug: 'cookbook', seoTitle: 'Go terminal UI recipes: the tui cookbook' },
	{ src: 'docs/program.md', slug: 'program' },
	{ src: "docs/layout.md", slug: "layout", seoTitle: 'Terminal UI layout in Go: rows, columns, grids | tui',
		description: "Lay out a Go terminal UI with layout.Node: two-pass measure and render, rows, columns, flex, grids, overlays, and how to place widgets in a layout." },
	{ src: "docs/widgets.md", slug: "widgets", seoTitle: 'Go terminal UI widgets: inputs, tables, trees, dialogs | tui',
		description: "Stateless widget functions and stateful component packages for Go terminal UIs: the shared contracts, and a catalog of every component with its example program." },
	{ src: 'docs/input.md', slug: 'input' },
	{ src: "docs/theming.md", slug: "theming",
		description: "Theme a Go terminal UI with theme.Theme: colour roles, preset themes, component tokens, glyph sets with an ASCII fallback, and colour depth." },
	{ src: 'docs/rendering.md', slug: 'rendering' },
	{ src: 'docs/capabilities.md', slug: 'capabilities' },
	{ src: 'docs/accessibility.md', slug: 'accessibility', seoTitle: 'Accessible terminal UIs in Go: screen readers and reduced motion | tui' },
	{ src: 'docs/migrating-to-v1.md', slug: 'migrating-to-v1', optional: true },
	{ src: 'docs/v1.md', slug: 'v1', optional: true },
	{ src: "docs/testing.md", slug: "testing", seoTitle: 'Testing Go terminal UIs: golden files and a headless harness | tui',
		description: "Run and write tests for tui: the tuitest headless harness, golden files, PTY tests, race, fuzz and benchmark runs, and the test environment variables." },
	{ src: 'docs/architecture/overview.md', slug: 'architecture', optional: true },
	{ src: 'CONTRIBUTING.md', slug: 'contributing', optional: true },
	{ src: 'CHANGELOG.md', slug: 'changelog', optional: true },
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

// describe builds a meta description from the first prose paragraphs: whole
// sentences until it has at least 90 characters, at most about 160.
function describe(body) {
	const paras = body
		.split(/\n\s*\n/)
		.map((s) => s.trim())
		.filter((s) => s && !/^(#|```|\||[-*] |>|<|\d+\. )/.test(s))
		.map(plainText)
		.filter((s) => !s.endsWith(':'));
	let out = '';
	// A sentence ends at . ! or ? followed by a space and a capital, so the dots
	// in identifiers (tui.NewProgram) and file names (go.mod) don't split it.
	for (const sentence of paras.join(' ').split(/(?<=[.!?])\s+(?=[A-Z])/)) {
		const next = (out + ' ' + sentence.trim()).trim();
		if (out && next.length > 165) break;
		out = next;
		if (out.length >= 90) break;
	}
	if (!out) return undefined;
	return out.length > 200 ? out.slice(0, 197) + '...' : out;
}

function frontmatter(fields) {
	const lines = Object.entries(fields)
		.filter(([, v]) => v !== undefined)
		.map(([k, v]) => `${k}: ${JSON.stringify(v)}`);
	return `---\n${lines.join('\n')}\n---\n\n`;
}

// pageHead returns the Starlight `head` entries for a generated page.
function pageHead({ slug, title, seoTitle, description }) {
	const article = {
		'@context': 'https://schema.org',
		'@type': 'TechArticle',
		headline: title,
		description,
		url: `${SITE}/${slug}/`,
		inLanguage: 'en',
		isPartOf: { '@id': `${SITE}/#website` },
		about: { '@id': `${SITE}/#software` },
	};
	return [
		{ tag: 'title', content: seoTitle ?? `${title} | ${TITLE_SUFFIX}` },
		{ tag: 'link', attrs: { rel: 'alternate', type: 'text/markdown', href: `/${slug}.md` } },
		{ tag: 'script', attrs: { type: 'application/ld+json' }, content: JSON.stringify(article) },
	];
}

// Plain Markdown copies of every page, for AI assistants (see llms.txt).
const mdCopies = [];

function addMarkdownCopy({ slug, title, description, body, source, optional }) {
	const absolute = body.replace(/\]\(\/(?!\/)/g, `](${SITE}/`);
	const text =
		`# ${title}\n\n` +
		(description ? `> ${description}\n\n` : '') +
		`Web: ${SITE}/${slug}/` +
		(source ? `  \nSource: ${GITHUB}/blob/${BRANCH}/${source}` : '') +
		`\n\n${absolute.trim()}\n`;
	fs.writeFileSync(path.join(publicDir, `${slug}.md`), text);
	mdCopies.push({ slug, title, description, text, optional });
}

function syncPage({ src, slug, title, seoTitle, description: given, optional }) {
	const md = fs.readFileSync(path.join(repoDir, src), 'utf8');
	const h1 = md.match(/^# (.+)\n/m);
	const raw = h1 ? md.slice(h1.index + h1[0].length) : md;
	const pageTitle = title ?? h1?.[1].trim() ?? slug;
	const description = given ?? describe(raw);
	const body = rewriteLinks(raw.replace(/^\n+/, ''), src);
	const out =
		frontmatter({
			title: pageTitle,
			description,
			editUrl: `${GITHUB}/edit/${BRANCH}/${src}`,
			head: pageHead({ slug, title: pageTitle, seoTitle, description }),
		}) + body;
	fs.mkdirSync(path.dirname(path.join(outDir, `${slug}.md`)), { recursive: true });
	fs.writeFileSync(path.join(outDir, `${slug}.md`), out);
	addMarkdownCopy({ slug, title: pageTitle, description, body, source: src, optional });
}

// writeLlmsTxt writes public/llms.txt (an index of the Markdown copies) and
// public/llms-full.txt (all of them in one file).
function writeLlmsTxt() {
	const readme = fs.readFileSync(path.join(repoDir, 'README.md'), 'utf8');
	const intro = readme
		.replace(/^# .+\n+/, '')
		.split(/\n\s*\n/)
		.slice(0, 2)
		.map((p) => plainText(p))
		.join('\n\n');
	const entry = (p) => `- [${p.title}](${SITE}/${p.slug}.md)` + (p.description ? `: ${p.description}` : '');
	let txt = `# tui\n\n> tui is a terminal UI (TUI) framework for Go, built on the Elm Architecture (Model, Update, View). Module path: github.com/ows4444/tui.\n\n`;
	txt += `${intro}\n\n`;
	txt += `Install with \`go get github.com/ows4444/tui\` (Go 1.26 or later). Source: ${GITHUB}. API reference: https://pkg.go.dev/github.com/ows4444/tui. License: MIT.\n\n`;
	txt += `## Docs\n\n${mdCopies.filter((p) => !p.optional).map(entry).join('\n')}\n\n`;
	txt += `## Optional\n\n${mdCopies.filter((p) => p.optional).map(entry).join('\n')}\n`;
	fs.writeFileSync(path.join(publicDir, 'llms.txt'), txt);
	fs.writeFileSync(path.join(publicDir, 'llms-full.txt'), mdCopies.map((p) => p.text).join('\n---\n\n'));
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

// exportStyledScreens runs the examples' size-matrix tests with
// TUI_SCREENS_DIR set, which makes testutil.SizeMatrix write each 80x24 frame
// with its colours (see internal/testutil/sizematrix.go), and returns them by
// example name. The environment is fixed so the frames don't depend on the
// machine. Without Go, or if the run fails, it returns {} and the screens are
// shown without colour.
function exportStyledScreens() {
	const dir = path.join(genDir, 'ansi');
	fs.rmSync(dir, { recursive: true, force: true });
	const env = { ...process.env, TUI_SCREENS_DIR: dir, TERM: 'xterm-256color', COLORTERM: 'truecolor', LANG: 'en_US.UTF-8' };
	delete env.NO_COLOR;
	delete env.LC_ALL;
	delete env.LC_CTYPE;
	const run = spawnSync('go', ['test', '-count=1', '-run', 'TestSizeMatrix/^80x24$', './examples/...'], {
		cwd: repoDir,
		env,
		encoding: 'utf8',
	});
	if (run.error || run.status !== 0) {
		console.warn(`sync-docs: could not export styled screens (${run.error?.message ?? (run.stdout + run.stderr).trim().split('\n').slice(-3).join(' | ')}); showing them without colour`);
		return {};
	}
	const out = {};
	for (const f of fs.existsSync(dir) ? fs.readdirSync(dir) : []) {
		if (f.endsWith('.ansi')) out[f.slice(0, -5)] = fs.readFileSync(path.join(dir, f), 'utf8');
	}
	return out;
}

const stripEscapes = (s) =>
	s.replace(/\x1b(?:\[[0-9;:?<=>]*[ -\/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/g, '');
const normalise = (s) =>
	s
		.split('\n')
		.map((l) => l.replace(/\r/g, '').trimEnd())
		.join('\n')
		.replace(/\s+$/, '');

const mdx = (s) => s.replace(/[{}]/g, (c) => `\\${c}`).replace(/</g, '&lt;');

// Categories for the Examples page, most visual first. An example in none of
// them is listed under "More", so a new example still appears.
const EXAMPLE_CATEGORIES = [
	{ id: 'showcase', title: 'Showcase', blurb: 'Full screens that show what a tui program can look like.', names: ['dashboard', 'canvas', 'chat', 'welcomescreen', 'splashscreen', 'probe', 'faces', 'avatar'] },
	{ id: 'forms', title: 'Forms and input', blurb: 'Text fields, sign-in and multi-step flows, focus and the cursor.', names: ['loginflow', 'setupflow', 'login', 'signup', 'form', 'inputs', 'controls', 'pickers', 'focus', 'cursorfield'] },
	{ id: 'lists', title: 'Lists, tables and navigation', blurb: 'Selection, tabs, scrolling and switching screens.', names: ['inspector', 'table', 'list', 'menus', 'panes', 'settings', 'router', 'pager'] },
	{ id: 'layout', title: 'Layout and rendering', blurb: 'Composing string and cell children, and the smallest program.', names: ['mixedscreen', 'counter'] },
	{ id: 'async', title: 'Async and streaming', blurb: 'Commands that resolve later, child processes and committed output.', names: ['asyncload', 'timers', 'procstream', 'buildlog', 'agentshell'] },
	{ id: 'inline', title: 'Inline mode', blurb: 'Programs that draw in the normal scrollback instead of the alternate screen; their first frame is small by design.', names: ['inlinespinners', 'inlinebuild', 'inlinechat', 'inlinetall'] },
];

function syncExamples() {
	const names = fs
		.readdirSync(path.join(repoDir, 'examples'), { withFileTypes: true })
		.filter((d) => d.isDirectory())
		.map((d) => d.name)
		.sort();
	const styled = exportStyledScreens();
	const screens = {};
	const withScreen = [];
	const without = [];
	let coloured = 0;
	for (const name of names) {
		const golden = path.join(repoDir, 'examples', name, SCREEN);
		const desc = firstSentence(packageDoc(name));
		if (fs.existsSync(golden)) {
			const text = fs.readFileSync(golden, 'utf8').replace(/\s+$/, '');
			screens[name] = { text, description: desc };
			// Use the styled frame only when it is the same screen the test
			// checks: stripped of escapes, it must equal the golden text.
			const ansi = styled[name];
			if (ansi !== undefined) {
				if (normalise(stripEscapes(ansi)) === normalise(text)) {
					screens[name].ansi = ansi;
					coloured++;
				} else {
					console.warn(`sync-docs: examples/${name}: styled frame differs from its golden; showing it without colour`);
				}
			}
			withScreen.push(name);
		} else {
			without.push(name);
		}
	}

	const meta = {
		slug: 'examples',
		title: 'Examples',
		seoTitle: 'Go terminal UI examples: 30 runnable tui programs with screens',
		description: 'Every program under examples/, with the 80×24 screen its test checks.',
	};
	const head = frontmatter({ title: meta.title, description: meta.description, editUrl: false, head: pageHead(meta) });
	// The page is MDX: examples are grouped by category (EXAMPLE_CATEGORIES)
	// and each is an ExampleCard drawing its real screen. The Markdown copy for
	// llms.txt gets the same structure with plain-text screens.
	const groups = EXAMPLE_CATEGORIES.map((c) => ({ ...c, names: c.names.filter((n) => withScreen.includes(n)) }));
	const unlisted = withScreen.filter((n) => !EXAMPLE_CATEGORIES.some((c) => c.names.includes(n)));
	if (unlisted.length) groups.push({ id: 'more', title: 'More', blurb: '', names: unlisted });
	const shown = groups.filter((g) => g.names.length);

	let page = `import ExampleCard from '../../components/ExampleCard.astro';\nimport ExampleFilter from '../../components/ExampleFilter.astro';\n\n`;
	let copy = '';
	const intro =
		`Every directory under [examples/](${GITHUB}/tree/${BRANCH}/examples) is a runnable program. ` +
		`Clone the repository and run one from its root, for example \`go run ./examples/dashboard\`.\n\n` +
		`The screens are not mock-ups: each is the first frame the example's own size-matrix test renders at 80×24, ` +
		`shown with the colours its \`View\` produces. Only the 16 basic terminal colours depend on a palette, as in any terminal.\n\n`;
	page += intro;
	copy += intro;
	page += `<ExampleFilter categories={${JSON.stringify(shown.map((g) => ({ id: g.id, title: g.title, count: g.names.length })))}} />\n\n`;
	for (const g of shown) {
		page += `<section class="ex-cat" data-cat="${g.id}">\n\n## ${g.title}\n\n` + (g.blurb ? `${g.blurb}\n\n` : '');
		copy += `## ${g.title}\n\n` + (g.blurb ? `${g.blurb}\n\n` : '');
		for (const name of g.names) {
			const { text, description } = screens[name];
			page += `<ExampleCard name="${name}" />\n\n`;
			copy += `### ${name}\n\n` + (description ? `${description}\n\n` : '') + `Run: \`go run ./examples/${name}\`\n\n`;
			copy += '````text\n' + text + '\n````\n\n';
			copy += `[Source](${GITHUB}/tree/${BRANCH}/examples/${name})\n\n`;
		}
		page += `</section>\n\n`;
	}
	if (without.length) {
		const list = without.map((name) => {
			const desc = firstSentence(packageDoc(name));
			return { name, desc };
		});
		page += `## Without a screen capture\n\n` + list.map(({ name, desc }) => `- [${name}](${GITHUB}/tree/${BRANCH}/examples/${name})` + (desc ? `: ${mdx(desc)}` : '')).join('\n') + '\n';
		copy += `## Without a screen capture\n\n` + list.map(({ name, desc }) => `- [${name}](${GITHUB}/tree/${BRANCH}/examples/${name})` + (desc ? `: ${desc}` : '')).join('\n') + '\n';
	}
	fs.writeFileSync(path.join(outDir, 'examples.mdx'), head + page);
	addMarkdownCopy({ ...meta, body: copy });
	fs.mkdirSync(genDir, { recursive: true });
	fs.writeFileSync(path.join(genDir, 'screens.json'), JSON.stringify(screens, null, '\t') + '\n');
	console.log(`sync-docs: ${coloured} of ${withScreen.length} example screens in colour`);
	return withScreen.length;
}

// Clear previously generated output, keeping the hand-written landing page and
// the committed files in public/.
for (const entry of fs.readdirSync(outDir)) {
	if (entry !== 'index.mdx') fs.rmSync(path.join(outDir, entry), { recursive: true });
}
for (const entry of fs.readdirSync(publicDir)) {
	if (entry.endsWith('.md') || /^llms(-full)?\.txt$/.test(entry)) fs.rmSync(path.join(publicDir, entry));
}
for (const page of pages) syncPage(page);
const n = syncExamples();
writeLlmsTxt();
console.log(`sync-docs: ${pages.length} pages, ${n} example screens, ${mdCopies.length} Markdown copies, llms.txt`);
if (broken) {
	console.error(`sync-docs: ${broken} broken link(s)`);
	process.exit(1);
}
