import type { SkinTone } from '@enzyme/shared';

const SKIN_TONE_KEY = 'enzyme:skin-tone';

export function getSavedSkinTone(): SkinTone {
  return (localStorage.getItem(SKIN_TONE_KEY) || '') as SkinTone;
}

export function saveSkinTone(tone: SkinTone): void {
  if (tone) {
    localStorage.setItem(SKIN_TONE_KEY, tone);
  } else {
    localStorage.removeItem(SKIN_TONE_KEY);
  }
}
