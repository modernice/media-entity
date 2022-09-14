import { ResponseOf } from '@modernice/typed-response'

/**
 * Image represents an image that may be stored in (cloud) storage.
 */
export interface Image<Languages extends string = string> {
  /**
   * Storage location of the image.
   */
  storage: ImageStorage

  /**
   * Filename of the image (without the directory).
   */
  filename: string

  /**
   * Filesize in bytes.
   */
  filesize: number

  /**
   * Width and height of the image.
   */
  dimensions: ImageDimensions

  /**
   * Localized names of the image.
   */
  names: { [lang in Languages]?: string }

  /**
   * Localized descriptions of the image.
   */
  descriptions: { [lang in Languages]?: string }
}

/**
 * ImageStorage provides the storage location of an {@link Image}.
 */
export interface ImageStorage {
  provider: string
  path: string
}

/**
 * ImageDimensions provides the width and height of an {@link Image}.
 */
export interface ImageDimensions {
  width: number
  height: number
}

/**
 * Hydrates an {@link Image} from an API response.
 */
export function hydrateImage<Languages extends string = string>(
  data: ResponseOf<Image<Languages>>,
  options?: { languages?: Languages[] }
): Image<Languages> {
  const nameKeys =
    options?.languages || (Object.keys(data.names as {}) as Languages[])
  const descKeys =
    options?.languages || (Object.keys(data.names as {}) as Languages[])

  const names = data.names as Record<Languages, string | undefined>
  const descriptions = data.descriptions as Record<
    Languages,
    string | undefined
  >

  return {
    ...data,
    names: nameKeys.reduce(
      (prev, lang) => ({ ...prev, [lang]: names[lang] }),
      {} as Record<Languages, string>
    ),
    descriptions: descKeys.reduce(
      (prev, lang) => ({ ...prev, [lang]: descriptions[lang] }),
      {} as Record<Languages, string>
    ),
  }
}
