/**
 * The catalogue of Verified Software Bases.
 *
 * Each Base is something I've actually built, tested, and would happily run
 * for someone else. Empty by default — paste a new one in below using the
 * example shape, then the site picks it up automatically.
 *
 * Mandatory fields per docs/PRD-full.md §14 (Listing Page Requirements):
 *   does[], doesNotDo[], limitations[], alternatives.{free, paidSaas,
 *   promptYourself, bestIf}, lastVerified, badges[], tiers[].
 */

export type Category =
  | 'mac-app'
  | 'website'
  | 'internal-tool'
  | 'creator-tool'
  | 'mobile-prototype';

export type Difficulty = 'easy' | 'medium' | 'hard';

export type Badge =
  | 'live-preview-verified'
  | 'source-build-verified'
  | 'remix-ready'
  | 'hosted-eligible'
  | 'mac-build-verified'
  | 'mobile-beta-available'
  | 'no-external-api'
  | 'local-first'
  | 'commercial-use-allowed';

export interface Tier {
  key: 'preview' | 'use' | 'own' | 'remix' | 'launch' | 'hosting';
  label: string;
  /** Display price, e.g. "$49" or "$19/mo". Authoritative price for checkout lives server-side. */
  price: string;
  includes: string[];
  note?: string;
}

export interface Base {
  slug: string;
  title: string;
  category: Category;
  oneLiner: string;
  description: string;
  /** Live preview URL (or screenshot/video URL for native apps). */
  previewUrl?: string;
  /** Optional video URL (mp4 or YouTube embed). */
  videoUrl?: string;
  does: string[];
  doesNotDo: string[];
  techStack: string[];
  badges: Badge[];
  limitations: string[];
  diy: {
    prompt: Difficulty;
    debug: Difficulty;
    launch: Difficulty;
    selfBuildHours: string;
    setupFromBase: string;
    aiOftenGetsWrong: string[];
    whyBuy: string;
  };
  alternatives: {
    free: string;
    paidSaas: string;
    promptYourself: string;
    bestIf: string;
  };
  tiers: Tier[];
  startingPrice: number;
  /** ISO date the Base was last hands-on verified. */
  lastVerified: string;
  creator: { name: string; handle?: string };
  license: string;
}

export const CATEGORY_META: Record<Category, { label: string; blurb: string }> = {
  'mac-app':            { label: 'Mac apps',        blurb: 'Local-first, downloadable, no hosting burden.' },
  'website':            { label: 'Websites',        blurb: 'Live preview, simple deploy, clear scope.' },
  'internal-tool':      { label: 'Internal tools',  blurb: 'Ownable workflows where SaaS is overkill.' },
  'creator-tool':       { label: 'Creator tools',   blurb: 'Small dashboards for operators and solo creators.' },
  'mobile-prototype':   { label: 'Mobile prototypes', blurb: 'TestFlight previews; source-ownable.' },
};

export const BADGE_META: Record<Badge, { label: string; tone: 'verified' | 'neutral' | 'accent' }> = {
  'live-preview-verified': { label: 'Live preview verified', tone: 'verified' },
  'source-build-verified': { label: 'Source build verified', tone: 'verified' },
  'remix-ready':           { label: 'Remix ready',           tone: 'accent' },
  'hosted-eligible':       { label: 'Hosted eligible',       tone: 'neutral' },
  'mac-build-verified':    { label: 'Mac build verified',    tone: 'verified' },
  'mobile-beta-available': { label: 'Mobile beta available', tone: 'neutral' },
  'no-external-api':       { label: 'No external API',       tone: 'neutral' },
  'local-first':           { label: 'Local-first',           tone: 'neutral' },
  'commercial-use-allowed':{ label: 'Commercial use ok',     tone: 'neutral' },
};

/**
 * Add your real Bases below. Reference shape (copy + edit):
 *
 *   {
 *     slug: 'your-base-slug',
 *     title: 'Your Base',
 *     category: 'mac-app',
 *     oneLiner: 'One sentence — the elevator pitch.',
 *     description: 'A paragraph or two of detail.',
 *     previewUrl: 'https://your.preview.url',
 *     does: ['Capability one', 'Capability two'],
 *     doesNotDo: ['Limitation one', 'Limitation two'],
 *     techStack: ['Astro', 'Tailwind v4'],
 *     badges: ['live-preview-verified', 'source-build-verified', 'remix-ready'],
 *     limitations: ['Tested up to X', 'Requires Y account'],
 *     diy: {
 *       prompt: 'medium',
 *       debug:  'medium',
 *       launch: 'medium',
 *       selfBuildHours: '6–12 hours',
 *       setupFromBase:  '20 minutes',
 *       aiOftenGetsWrong: ['Subtle bug 1', 'Subtle bug 2'],
 *       whyBuy: 'Short reason for paying instead of prompting.',
 *     },
 *     alternatives: {
 *       free:           'Free option to consider.',
 *       paidSaas:       'Mature SaaS alternative.',
 *       promptYourself: 'When prompting it yourself is the right call.',
 *       bestIf:         'When this Base is the right call.',
 *     },
 *     tiers: [
 *       { key: 'preview', label: 'Live preview', price: 'Free', includes: ['Working demo'] },
 *       { key: 'use',     label: 'Use it',       price: '$19',  includes: ['Packaged build'] },
 *       { key: 'own',     label: 'Own it',       price: '$79',  includes: ['GitHub source', 'Commercial license'] },
 *     ],
 *     startingPrice: 19,
 *     lastVerified: '2026-06-15',
 *     creator: { name: 'Sarthak Agrawal', handle: 'sarthakagrawal927' },
 *     license: 'Buyer owns their copy. Modify + use commercially. No resale of unchanged source.',
 *   }
 *
 * Once you add a Base, also configure its pricing on the BE
 * (`api/handlers/prices.go`) and the per-tier Dodo product IDs.
 */
export const BASES: Base[] = [];

export function basesByCategory(category: Category): Base[] {
  return BASES.filter(b => b.category === category);
}

/**
 * Returns up to six Bases for the homepage Featured grid.
 * Order is curated by hand: tweak as you grow.
 */
export function featuredBases(): Base[] {
  return BASES.slice(0, 6);
}
