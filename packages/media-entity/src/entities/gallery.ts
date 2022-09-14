import {
  hydrateImage,
  Image as BaseImage,
  type ImageDimensions,
} from '../image'
import { ResponseOf } from '@modernice/typed-response'

/**
 * Gallery is an image gallery. Each image in the gallery is represented as a
 * {@link Stack}, which can contain multiple variants of the same image (for
 * example in different {@link ImageDimensions|dimensions}).
 */
export interface Gallery<Languages extends string = string> {
  stacks: Stack<Languages>[]
}

/**
 * A Stack is a collection of {@link Image|images} that are variants of the same image.
 */
export interface Stack<Languages extends string = string> {
  id: string
  variants: Image<Languages>[]
  tags: string[]
}

/**
 * Image is an image variant within a {@link Stack}. The original image of the
 * Stack has its `original` field is set to `true`.
 */
export interface Image<Languages extends string = string>
  extends BaseImage<Languages> {
  id: string
  original: boolean
}

/**
 * Hydrate a {@link Gallery} from an API response.
 */
export function hydrateGallery<Languages extends string = string>(
  data: ResponseOf<Gallery<Languages>>
): Gallery<Languages> {
  return {
    ...data,
    stacks: data.stacks.map((data) => hydrateStack(data)),
  }
}

/**
 * Hydrates a {@link Stack} from an API response.
 */
export function hydrateStack<Languages extends string = string>(
  data: ResponseOf<Stack<Languages>>
): Stack<Languages> {
  return {
    ...data,
    variants: data.variants.map(hydrateStackImage),
  }
}

/**
 * Hydrates an {@link Image} from an API response.
 */
export function hydrateStackImage<Languages extends string = string>(
  data: ResponseOf<Image<Languages>>
): Image<Languages> {
  return {
    ...data,
    ...hydrateImage(data),
  }
}
