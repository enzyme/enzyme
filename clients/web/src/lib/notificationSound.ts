let audioContext: AudioContext | null = null;
let audioUnlocked = false;

// Initialize AudioContext on first user interaction
function getAudioContext(): AudioContext | null {
  if (!audioContext) {
    try {
      audioContext = new AudioContext();
    } catch {
      return null;
    }
  }
  return audioContext;
}

// Unlock audio context on user interaction (required by browsers)
export function unlockAudio() {
  if (audioUnlocked) return;

  const ctx = getAudioContext();
  if (ctx && ctx.state === 'suspended') {
    ctx
      .resume()
      .then(() => {
        audioUnlocked = true;
      })
      .catch(() => {
        // Will retry on next interaction
      });
  } else if (ctx) {
    audioUnlocked = true;
  }
}

// Play a two-tone notification chime using Web Audio API
export function playNotificationSound() {
  const ctx = getAudioContext();
  if (!ctx || !audioUnlocked) return;

  // Resume context if suspended
  if (ctx.state === 'suspended') {
    ctx.resume().catch(() => {});
  }

  const now = ctx.currentTime;
  const duration = 0.15;
  const gap = 0.05;
  const volume = 0.3;

  // First tone: D5 (~587 Hz)
  playTone(ctx, 587, now, duration, volume);
  // Second tone: A5 (~880 Hz)
  playTone(ctx, 880, now + duration + gap, duration, volume);
}

function playTone(
  ctx: AudioContext,
  frequency: number,
  startTime: number,
  duration: number,
  volume: number,
) {
  const oscillator = ctx.createOscillator();
  const gainNode = ctx.createGain();

  oscillator.type = 'sine';
  oscillator.frequency.setValueAtTime(frequency, startTime);

  // Envelope: quick attack, short sustain, smooth decay
  gainNode.gain.setValueAtTime(0, startTime);
  gainNode.gain.linearRampToValueAtTime(volume, startTime + 0.01);
  gainNode.gain.setValueAtTime(volume, startTime + duration * 0.6);
  gainNode.gain.exponentialRampToValueAtTime(0.001, startTime + duration);

  oscillator.connect(gainNode);
  gainNode.connect(ctx.destination);

  oscillator.start(startTime);
  oscillator.stop(startTime + duration);
}

// Request browser notification permission
export async function requestNotificationPermission(): Promise<boolean> {
  if (!('Notification' in window)) {
    return false;
  }

  if (Notification.permission === 'granted') {
    return true;
  }

  if (Notification.permission === 'denied') {
    return false;
  }

  const permission = await Notification.requestPermission();
  return permission === 'granted';
}

// Show browser notification
export function showBrowserNotification(title: string, body: string, onClick?: () => void) {
  if (!('Notification' in window) || Notification.permission !== 'granted') {
    return;
  }

  const notification = new Notification(title, {
    body,
    icon: '/favicon.ico',
    tag: 'feather-notification',
  });

  if (onClick) {
    notification.onclick = () => {
      window.focus();
      onClick();
      notification.close();
    };
  }

  // Auto-close after 5 seconds
  setTimeout(() => notification.close(), 5000);
}
