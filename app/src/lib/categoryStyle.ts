import type { Category } from '@/api/types'

/** 24-unit stroke glyphs drawn white on a coloured circle. */
const ICONS = {
  cart: 'M3 3h2l2.4 12.2a1 1 0 0 0 1 .8h9.2a1 1 0 0 0 1-.8L20 7H6m3 13h.01M17 20h.01',
  cup: 'M4 8h13v5a5 5 0 0 1-5 5H9a5 5 0 0 1-5-5zm13 1h1.5a2.5 2.5 0 0 1 0 5H17M8 2v3m4-3v3',
  car: 'M5 17h14v-5l-2-5H7l-2 5zm0 0v2m14-2v2M5 12h14m-11 2.5h.01m7.99 0h.01',
  bus: 'M6 3h12a2 2 0 0 1 2 2v11H4V5a2 2 0 0 1 2-2zM4 11h16M4 16h16v3H4zm3 3v2m10-2v2',
  fuel: 'M4 21V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v16M3 21h12M4 10h10m0-3 3 3v7a1.5 1.5 0 0 0 3 0V9l-3-3',
  bolt: 'M13 2 4 14h7l-1 8 9-12h-7z',
  house: 'M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z',
  phone: 'M8 2h8a2 2 0 0 1 2 2v16a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm3 16h2',
  cross: 'M9 3h6v6h6v6h-6v6H9v-6H3V9h6z',
  shirt: 'M8 3 4 6l2 4 2-1v12h8V9l2 1 2-4-4-3c-.5 1.5-2 2.5-4 2.5S8.5 4.5 8 3z',
  book: 'M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2zm0 16a2 2 0 0 1 2-2h13',
  play: 'M5 4h14a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zm5 5v6l5-3z',
  gift: 'M4 9h16v4H4zm1 4h14v8H5zM12 9v12m0-12c-2-4-6-4-6-1.5S9 9 12 9zm0 0c2-4 6-4 6-1.5S15 9 12 9z',
  users: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm13 10v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8',
  sparkle: 'M12 3l1.8 4.7 4.7 1.8-4.7 1.8L12 16l-1.8-4.7-4.7-1.8 4.7-1.8zm7 12 .8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8z',
  sofa: 'M4 11V8a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v3M2 13a2 2 0 0 1 4 0v2h12v-2a2 2 0 0 1 4 0v5H2zm3 5v2m14-2v2',
  plane: 'M2 13 22 5l-8 17-2-7z',
  bank: 'M3 10 12 4l9 6M5 10v8m4-8v8m6-8v8m4-8v8M3 20h18',
  box: 'M3 7l9-4 9 4v10l-9 4-9-4zm0 0 9 4 9-4m-9 4v10',
  briefcase: 'M4 7h16a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1zm5 0V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2M3 13h18',
  laptop: 'M5 5h14a1 1 0 0 1 1 1v9H4V6a1 1 0 0 1 1-1zM2 19h20',
  card: 'M3 6h18a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1zM2 10h20M6 15h4',
  coin: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zm3-12.5c-.5-1-1.6-1.5-3-1.5-2 0-3 1-3 2.3 0 3 6 1.7 6 4.7 0 1.3-1.2 2.5-3 2.5-1.5 0-2.6-.6-3-1.6M12 5.5V7m0 10v1.5',
  tag: 'M3 12V4a1 1 0 0 1 1-1h8l9 9-9 9zm4.5-4.5h.01',
} as const

export interface CategoryStyle {
  color: string
  path: string
}

/** One colour and glyph per default category key the backend seeds. */
export const CATEGORY_STYLES: Record<string, CategoryStyle> = {
  groceries: { color: '#2bb673', path: ICONS.cart },
  cafe: { color: '#f08a24', path: ICONS.cup },
  taxi: { color: '#f5b400', path: ICONS.car },
  transport: { color: '#3b82f6', path: ICONS.bus },
  fuel: { color: '#64748b', path: ICONS.fuel },
  utilities: { color: '#eab308', path: ICONS.bolt },
  rent: { color: '#7c5cff', path: ICONS.house },
  mobile_internet: { color: '#06b6d4', path: ICONS.phone },
  health: { color: '#ef4444', path: ICONS.cross },
  clothing: { color: '#ec4899', path: ICONS.shirt },
  education: { color: '#4f46e5', path: ICONS.book },
  entertainment: { color: '#a855f7', path: ICONS.play },
  gifts: { color: '#f43f5e', path: ICONS.gift },
  family: { color: '#14b8a6', path: ICONS.users },
  beauty: { color: '#d946ef', path: ICONS.sparkle },
  home: { color: '#a16207', path: ICONS.sofa },
  travel: { color: '#0ea5e9', path: ICONS.plane },
  fees: { color: '#475569', path: ICONS.bank },
  other_expense: { color: '#8b8aa0', path: ICONS.box },
  salary: { color: '#10b981', path: ICONS.briefcase },
  freelance: { color: '#22c55e', path: ICONS.laptop },
  gift_income: { color: '#f43f5e', path: ICONS.gift },
  cashback: { color: '#5b3df5', path: ICONS.card },
  other_income: { color: '#0f9e6e', path: ICONS.coin },
}

/** Colours for custom categories, picked stably by id. */
const CUSTOM_COLORS = ['#5b3df5', '#2bd9a0', '#ff6b5a', '#f5a524', '#3b82f6', '#ec4899', '#14b8a6', '#a855f7']

export function categoryStyle(cat: Pick<Category, 'id' | 'key'> | undefined): CategoryStyle {
  if (cat?.key && CATEGORY_STYLES[cat.key]) return CATEGORY_STYLES[cat.key]!
  const id = cat?.id ?? 0
  return { color: CUSTOM_COLORS[Math.abs(id) % CUSTOM_COLORS.length]!, path: ICONS.tag }
}
