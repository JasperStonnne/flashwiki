/**
 * Type definitions for Strapi media response.
 * Used for future integration with Strapi CMS.
 */

export interface StrapiMediaFormat {
  name: string;
  hash: string;
  ext: string;
  mime: string;
  width: number;
  height: number;
  size: number;
  url: string;
}

export interface StrapiMediaFormats {
  thumbnail?: StrapiMediaFormat;
  small?: StrapiMediaFormat;
  medium?: StrapiMediaFormat;
  large?: StrapiMediaFormat;
}

export interface StrapiMedia {
  id: number;
  name: string;
  alternativeText: string | null;
  caption: string | null;
  width: number;
  height: number;
  formats: StrapiMediaFormats | null;
  hash: string;
  ext: string;
  mime: string;
  size: number;
  url: string;
  previewUrl: string | null;
  provider: string;
  createdAt: string;
  updatedAt: string;
}

export interface StrapiMediaResponse {
  data: {
    id: number;
    attributes: StrapiMedia;
  };
}

export interface StrapiMediaListResponse {
  data: Array<{
    id: number;
    attributes: StrapiMedia;
  }>;
}
