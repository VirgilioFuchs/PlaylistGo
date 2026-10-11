import type { Thumbnail, ThumbnailVariants, TrackSnapshot } from "../types";

const thumbnailPriority: Array<keyof ThumbnailVariants> = ["maxres", "standard", "high", "medium", "default"];
const thumbnailByAscendingWidth: Array<keyof ThumbnailVariants> = ["default", "medium", "high", "standard", "maxres"];

export function getThumbnailForWidth(
  track: Pick<TrackSnapshot, "thumbnailUrl" | "thumbnails">,
  cssWidth: number,
  devicePixelRatio = getCappedDevicePixelRatio()
): Thumbnail | null {
  const thumbnails = track.thumbnails;
  const requiredWidth = Math.max(1, cssWidth) * devicePixelRatio;

  if (thumbnails) {
    for (const key of thumbnailByAscendingWidth) {
      const thumbnail = getValidThumbnail(thumbnails[key]);
      if (thumbnail && thumbnail.width !== undefined && thumbnail.width >= requiredWidth) {
        return thumbnail;
      }
    }

    const bestAvailable = getBestThumbnail(thumbnails);
    if (bestAvailable) {
      return bestAvailable;
    }
  }

  const fallbackUrl = sanitizeThumbnailUrl(track.thumbnailUrl);
  return fallbackUrl ? { url: fallbackUrl } : null;
}

export function getBestThumbnail(thumbnails: ThumbnailVariants): Thumbnail | null {
  for (const key of thumbnailPriority) {
    const thumbnail = getValidThumbnail(thumbnails[key]);
    if (thumbnail) {
      return thumbnail;
    }
  }
  return null;
}

export function sanitizeThumbnailUrl(thumbnailUrl: string | undefined): string {
  if (!thumbnailUrl) {
    return "";
  }

  const normalized = thumbnailUrl.trim();
  if (
    normalized.startsWith("http://") ||
    normalized.startsWith("https://") ||
    normalized.startsWith("data:") ||
    normalized.startsWith("blob:") ||
    normalized.startsWith("/")
  ) {
    return normalized;
  }

  return "";
}

function getValidThumbnail(thumbnail: Thumbnail | undefined): Thumbnail | null {
  const url = sanitizeThumbnailUrl(thumbnail?.url);
  if (!url) {
    return null;
  }

  return {
    url,
    ...(thumbnail?.width && thumbnail.width > 0 ? { width: thumbnail.width } : {}),
    ...(thumbnail?.height && thumbnail.height > 0 ? { height: thumbnail.height } : {})
  };
}

function getCappedDevicePixelRatio(): number {
  if (typeof window === "undefined") {
    return 1;
  }

  // Above 2x, image bytes grow faster than the visible gain in dense UI.
  return Math.min(Math.max(window.devicePixelRatio || 1, 1), 2);
}
