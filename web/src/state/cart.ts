import type { TrackSnapshot } from "../types";
import { sanitizeThumbnailUrl } from "../lib/thumbnail";

export const CART_STORAGE_KEY = "musicgo.cart.v1";

export function normalizeCartTracks(tracks: TrackSnapshot[]): TrackSnapshot[] {
  const seen = new Set<string>();
  const unique: TrackSnapshot[] = [];

  for (const track of tracks) {
    if (seen.has(track.videoId)) {
      continue;
    }
    seen.add(track.videoId);
    unique.push({
      ...track,
      thumbnailUrl: sanitizeThumbnailUrl(track.thumbnailUrl),
      thumbnails: sanitizeThumbnailVariants(track.thumbnails)
    });
  }

  return unique;
}

function sanitizeThumbnailVariants(thumbnails: TrackSnapshot["thumbnails"]): TrackSnapshot["thumbnails"] {
  if (!thumbnails) {
    return undefined;
  }

  const sanitized: NonNullable<TrackSnapshot["thumbnails"]> = {};
  for (const key of Object.keys(thumbnails) as Array<keyof NonNullable<TrackSnapshot["thumbnails"]>>) {
    const thumbnail = thumbnails[key];
    const url = sanitizeThumbnailUrl(thumbnail?.url);
    if (url) {
      sanitized[key] = { ...thumbnail, url };
    }
  }

  return Object.keys(sanitized).length > 0 ? sanitized : undefined;
}

export function toggleTrackSelection(tracks: TrackSnapshot[], track: TrackSnapshot): TrackSnapshot[] {
  const index = tracks.findIndex((item) => item.videoId === track.videoId);
  if (index >= 0) {
    return tracks.filter((item) => item.videoId !== track.videoId);
  }

  return normalizeCartTracks([...tracks, track]);
}

export function removeTrackFromSelection(tracks: TrackSnapshot[], videoId: string): TrackSnapshot[] {
  return tracks.filter((item) => item.videoId !== videoId);
}

/**
 * Returns the current snapshots in the requested order when the requested IDs
 * form the exact, unique set currently selected. Invalid orders return null so
 * callers retain their canonical selection instead of partially reordering it.
 */
export function reorderTrackSelection(
  tracks: TrackSnapshot[],
  orderedVideoIds: string[]
): TrackSnapshot[] | null {
  if (orderedVideoIds.length !== tracks.length) {
    return null;
  }

  const tracksById = new Map<string, TrackSnapshot>();
  for (const track of tracks) {
    if (!track.videoId || tracksById.has(track.videoId)) {
      return null;
    }
    tracksById.set(track.videoId, track);
  }

  const requestedIds = new Set<string>();
  const orderedTracks: TrackSnapshot[] = [];
  for (const videoId of orderedVideoIds) {
    if (!videoId || requestedIds.has(videoId)) {
      return null;
    }

    const track = tracksById.get(videoId);
    if (!track) {
      return null;
    }

    requestedIds.add(videoId);
    orderedTracks.push(track);
  }

  return orderedTracks;
}

export function clearCartSelection(): TrackSnapshot[] {
  return [];
}

export function countSelectedTracks(tracks: TrackSnapshot[]): number {
  return tracks.length;
}

export function isTrackSelected(tracks: TrackSnapshot[], videoId: string): boolean {
  return tracks.some((item) => item.videoId === videoId);
}

export function loadCartSelection(): TrackSnapshot[] {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const raw = window.localStorage.getItem(CART_STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as TrackSnapshot[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return normalizeCartTracks(
      parsed.filter((item): item is TrackSnapshot => Boolean(item && item.videoId))
    );
  } catch {
    return [];
  }
}

export function saveCartSelection(tracks: TrackSnapshot[]): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(CART_STORAGE_KEY, JSON.stringify(normalizeCartTracks(tracks)));
  } catch {
    // Ignore storage failures. Cart state remains in memory.
  }
}
