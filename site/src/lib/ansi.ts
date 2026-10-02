// Parses a terminal frame (text with ANSI escape sequences, as a tui View
// returns it) into lines of characters, each with the SGR style in effect.
// Only SGR (ESC [ ... m) changes the style; other CSI, OSC (hyperlinks) and
// ESC sequences are skipped, as they don't change what a cell shows.

export interface Style {
	fg?: string; // CSS colour
	bg?: string;
	bold?: boolean;
	dim?: boolean;
	italic?: boolean;
	underline?: boolean;
	reverse?: boolean;
	strike?: boolean;
}

export interface Cell {
	ch: string;
	style: Style;
}

// The 16 standard colours are CSS variables (--ansi-0 … --ansi-15, set in
// custom.css), so the site chooses the palette, like a terminal theme does.
const ansi16 = (n: number) => `var(--ansi-${n})`;

function xterm256(n: number): string {
	if (n < 16) return ansi16(n);
	if (n < 232) {
		const v = [0, 95, 135, 175, 215, 255];
		const i = n - 16;
		return rgb(v[Math.floor(i / 36)], v[Math.floor(i / 6) % 6], v[i % 6]);
	}
	const g = 8 + (n - 232) * 10;
	return rgb(g, g, g);
}

const rgb = (r: number, g: number, b: number) =>
	`#${[r, g, b].map((x) => Math.max(0, Math.min(255, x)).toString(16).padStart(2, '0')).join('')}`;

// extended reads a 38/48 colour from params starting at i (pointing at 38/48);
// it returns the colour and how many params it used. Accepts both ";" and ":"
// separated forms, including 38:2::r:g:b with an empty colour-space id.
function extended(params: string[], i: number): [string | undefined, number] {
	const head = params[i];
	if (head.includes(':')) {
		const p = head.split(':');
		if (p[1] === '5') return [xterm256(Number(p[2])), 1];
		if (p[1] === '2') {
			const nums = p.slice(2).filter((s) => s !== '');
			const [r, g, b] = nums.slice(-3).map(Number);
			return [rgb(r, g, b), 1];
		}
		return [undefined, 1];
	}
	if (params[i + 1] === '5') return [xterm256(Number(params[i + 2])), 3];
	if (params[i + 1] === '2') {
		return [rgb(Number(params[i + 2]), Number(params[i + 3]), Number(params[i + 4])), 5];
	}
	return [undefined, 1];
}

function applySGR(style: Style, raw: string): Style {
	const params = raw === '' ? ['0'] : raw.split(';');
	let s = { ...style };
	for (let i = 0; i < params.length; ) {
		const code = Number(params[i].split(':')[0]);
		if (code === 38 || code === 48) {
			const [colour, used] = extended(params, i);
			if (code === 38) s.fg = colour;
			else s.bg = colour;
			i += used;
			continue;
		}
		switch (true) {
			case code === 0: s = {}; break;
			case code === 1: s.bold = true; break;
			case code === 2: s.dim = true; break;
			case code === 3: s.italic = true; break;
			case code === 4: s.underline = !params[i].endsWith(':0'); break;
			case code === 7: s.reverse = true; break;
			case code === 9: s.strike = true; break;
			case code === 22: s.bold = false; s.dim = false; break;
			case code === 23: s.italic = false; break;
			case code === 24: s.underline = false; break;
			case code === 27: s.reverse = false; break;
			case code === 29: s.strike = false; break;
			case code >= 30 && code <= 37: s.fg = ansi16(code - 30); break;
			case code === 39: s.fg = undefined; break;
			case code >= 40 && code <= 47: s.bg = ansi16(code - 40); break;
			case code === 49: s.bg = undefined; break;
			case code >= 90 && code <= 97: s.fg = ansi16(code - 90 + 8); break;
			case code >= 100 && code <= 107: s.bg = ansi16(code - 100 + 8); break;
		}
		i++;
	}
	return s;
}

// Matches one escape sequence: SGR (captured), any other CSI, an OSC ended by
// BEL or ST, or a two-character ESC sequence.
const ESC = /\x1b(?:\[([0-9;:]*)m|\[[0-9;:?<=>]*[ -\/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/y;

export function parseFrame(frame: string): Cell[][] {
	const lines: Cell[][] = [[]];
	let style: Style = {};
	let i = 0;
	while (i < frame.length) {
		if (frame[i] === '\x1b') {
			ESC.lastIndex = i;
			const m = ESC.exec(frame);
			if (m) {
				if (m[1] !== undefined) style = applySGR(style, m[1]);
				i = ESC.lastIndex;
				continue;
			}
			i++; // a lone ESC: drop it
			continue;
		}
		const cp = frame.codePointAt(i)!;
		const ch = String.fromCodePoint(cp);
		i += ch.length;
		if (ch === '\n') {
			lines.push([]);
			continue;
		}
		if (ch === '\r') continue;
		lines[lines.length - 1].push({ ch, style });
	}
	return lines;
}

// plainText is the frame with all escape sequences removed, for comparing
// against the golden file.
export function plainText(lines: Cell[][]): string {
	return lines.map((l) => l.map((c) => c.ch).join('')).join('\n');
}
