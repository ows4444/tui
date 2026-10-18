// Numbers the pages quote, read from the repository when the site is built, so
// they cannot go stale.
import fs from 'node:fs';
import path from 'node:path';

// The build runs in site/, so the repository is its parent.
const repo = path.resolve(process.cwd(), '..');

const dirs = (p) => fs.readdirSync(p, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name);
const hasGo = (p) => fs.readdirSync(p).some((f) => f.endsWith('.go') && !f.endsWith('_test.go'));

// Directories at the root that are not library packages.
const notLibrary = new Set(['cmd', 'examples', 'internal', 'site', 'docs', 'bench', 'testdata']);

const goMod = fs.readFileSync(path.join(repo, 'go.mod'), 'utf8');

export const facts = {
	go: /^go (\d+\.\d+)/m.exec(goMod)?.[1] ?? '1.26',
	// Programs with a screen on the Examples page: those whose test keeps a first frame.
	examples: dirs(path.join(repo, 'examples')).filter((d) => fs.existsSync(path.join(repo, 'examples', d, 'testdata', 'size-80x24.golden'))).length,
	packages: dirs(repo).filter((d) => !notLibrary.has(d) && !d.startsWith('.') && hasGo(path.join(repo, d))).length,
};
