/**
 * Anything that provides tags.
 */
export interface Taggable {
  tags: string[]
}

/**
 * Returns whether `v` has the given `tag`.
 */
export function hasTag(v: Taggable, tag: string) {
  return v.tags.includes(tag)
}
