/** Canonical English body-region taxonomy (mirrors generator/bodyparts). */

export const MAX_REGIONS = 9;

export const CANONICAL = [
  { id: 'face', label: 'Face', aliases: ['kopf', 'gesicht'] },
  { id: 'mouth', label: 'Mouth', aliases: ['mund', 'lips'] },
  { id: 'breasts', label: 'Breasts', aliases: ['brust', 'breast', 'tits'] },
  { id: 'nipples', label: 'Nipples', aliases: ['nipple', 'nippel', 'brustwarze', 'brustwarzen'] },
  { id: 'hand_1', label: 'Hand 1', aliases: ['hand', 'hand1', 'left_hand'] },
  { id: 'hand_2', label: 'Hand 2', aliases: ['hand2', 'right_hand'] },
  { id: 'penis', label: 'Penis', aliases: [] },
  { id: 'glans', label: 'Glans', aliases: ['eichel', 'tip'] },
  { id: 'vagina', label: 'Vagina', aliases: ['pussy', 'vulva'] },
];

export const CLASS_PRESETS = CANONICAL.map(p => p.id);

/** Zone 2 / contact proposals — body-part first (not generic "Partner"). */
export const CONTACT_CLASS_ORDER = [
  'mouth', 'hand_1', 'hand_2', 'vagina', 'nipples', 'breasts', 'face',
];

/** Zone 1 tip proposals — glans/penis preferred for Everyday CSRT. */
export const TIP_CLASS_ORDER = [
  'glans', 'penis', 'hand_1', 'hand_2',
];

const ALIAS = (() => {
  const m = Object.create(null);
  for (const p of CANONICAL) {
    m[p.id] = p.id;
    for (const a of p.aliases || []) m[String(a).toLowerCase()] = p.id;
  }
  return m;
})();

/** Map free-text / German legacy labels to canonical English IDs. */
export function normalizeClass(name) {
  const s = String(name || '').trim().toLowerCase().replace(/[\s-]+/g, '_');
  if (!s) return '';
  return ALIAS[s] || s;
}

export function preferredClassesCSV() {
  return CLASS_PRESETS.join(',');
}

export function labelFor(id) {
  const n = normalizeClass(id);
  const hit = CANONICAL.find(p => p.id === n);
  return hit ? hit.label : (id || '');
}

/** Ordered CANONICAL entries for a select; leftovers appended in taxonomy order. */
export function orderedCanonical(preferIds) {
  const seen = new Set();
  const out = [];
  for (const id of preferIds || []) {
    const n = normalizeClass(id);
    const hit = CANONICAL.find(p => p.id === n);
    if (hit && !seen.has(hit.id)) {
      seen.add(hit.id);
      out.push(hit);
    }
  }
  for (const p of CANONICAL) {
    if (!seen.has(p.id)) out.push(p);
  }
  return out;
}

export function defaultRole(classId) {
  switch (normalizeClass(classId)) {
    case 'penis': case 'glans': case 'hand_1': case 'hand_2':
      return 'tracked';
    case 'nipples': case 'mouth': case 'vagina': case 'breasts':
      return 'fixed';
    case 'face':
      return 'mask';
    default:
      return 'tracked';
  }
}
