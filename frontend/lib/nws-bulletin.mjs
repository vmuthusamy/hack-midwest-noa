// Regroups NWS alert text for display. Words are never rewritten: every string
// returned here is a substring of the original description, with hard line wraps joined.

const LABELED = /^([A-Z][A-Z /]+?)\.\.\.\s*(.*)$/;
const TAG = /^(HAZARD|SOURCE|IMPACT)\.\.\.\s*(.*)$/;
const PRODUCT = /^[A-Z0-9]{5,9}$/;              // AWIPS id, e.g. FFWCAE
const LOCATIONS = /locations? .*include/i;

const title = s => s.charAt(0) + s.slice(1).toLowerCase();
const join = lines => lines.join(' ').replace(/\s+/g, ' ').trim();
const trimDots = s => s.replace(/\.{3}$/, '').trim();

function items(lines) {
  // "- " starts an item; other lines continue the current one.
  const out = [];
  for (const line of lines) {
    if (line.startsWith('- ')) out.push(line.slice(2));
    else if (out.length) out[out.length - 1] += ' ' + line;
    else out.push(line);
  }
  return out.map(s => s.replace(/\s+/g, ' ').trim()).filter(Boolean);
}

export function splitPlaces(text) {
  return text.replace(/\.$/, '').split(/,\s*(?:and\s+)?|\s+and\s+/).map(s => s.trim()).filter(Boolean);
}

export function parseBulletin(description = '') {
  const result = { productId: null, sections: [], tags: {} };
  const paragraphs = String(description).replace(/\r/g, '').split(/\n\s*\n/)
    .map(p => p.split('\n').map(l => l.trim()).filter(Boolean)).filter(p => p.length);

  for (const lines of paragraphs) {
    const [first, ...rest] = lines;
    if (lines.length === 1 && PRODUCT.test(first)) { result.productId = first; continue; }
    const tag = TAG.exec(first);
    if (tag) { result.tags[tag[1].toLowerCase()] = join([tag[2], ...rest]); continue; }
    if (!first.startsWith('* ')) {
      const prev = result.sections.at(-1);
      // "This includes Interstate 20…" belongs to the location list above it.
      if (prev?.kind === 'locations' && /^This includes/.test(first)) prev.note = join(lines);
      else result.sections.push({ kind: 'text', text: join(lines) });
      continue;
    }
    const body = [first.slice(2), ...rest];
    const labeled = LABELED.exec(body[0]);
    if (labeled && labeled[1].length <= 24) {
      const content = [labeled[2], ...body.slice(1)].filter(Boolean);
      const list = content.some(l => l.startsWith('- '));
      result.sections.push(list
        ? { kind: 'labeled', label: title(labeled[1]), items: items(content) }
        : { kind: 'labeled', label: title(labeled[1]), text: join(content) });
    } else if (body[0].endsWith('...') && body.length > 1 && LOCATIONS.test(body[0])) {
      const text = join(body.slice(1));
      result.sections.push({ kind: 'locations', heading: trimDots(body[0]), places: splitPlaces(text) });
    } else if (body[0].endsWith('...') && body.length > 1) {
      result.sections.push({ kind: 'list', heading: trimDots(body[0]), items: body.slice(1).map(trimDots) });
    } else {
      result.sections.push({ kind: 'text', text: join(body), bullet: true });
    }
  }
  return result;
}

// Exact, case-insensitive match of a facility's place name against NWS-listed places.
export function listedPlaces(bulletin, names = []) {
  const wanted = names.filter(Boolean).map(n => n.toLowerCase());
  return bulletin.sections.filter(s => s.kind === 'locations')
    .flatMap(s => s.places).filter(p => wanted.includes(p.toLowerCase()));
}
