/** One point of a skill's difficulty curve (GET /api/osu/songs/{id}/strains). */
export interface StrainPoint {
  /** ms on the timeline the skills are calculated on; multiply by `speed` for the beatmap's own time. */
  time: number
  /** On the scale of the skill: a curve's maximum is the skill shown in the card. */
  value: number
}

export const SKILL_KEYS = [
  'stamina',
  'tenacity',
  'agility',
  'precision',
  'reading',
  'memory',
  'accuracy',
  'reaction',
] as const

export type SkillKey = (typeof SKILL_KEYS)[number]

export type StrainsResponse = {
  beatmap_id: number
  mods: string
  speed: number
  window: number
} & Record<SkillKey, StrainPoint[]>
