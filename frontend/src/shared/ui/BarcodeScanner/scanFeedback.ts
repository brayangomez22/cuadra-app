// A short beep and a vibration when a code is read, as a counter reader does:
// the cashier knows it was read without looking at the screen. Either may be
// missing (no vibration on iPhone or a PC): then it is skipped.
const BEEP = { hz: 1800, seconds: 0.08, volume: 0.1 };

let audio: AudioContext | undefined;

export function scanFeedback() {
  if ('vibrate' in navigator) navigator.vibrate(60);
  beep();
}

function beep() {
  if (typeof AudioContext === 'undefined') return;
  audio ??= new AudioContext();
  const oscillator = audio.createOscillator();
  const gain = audio.createGain();
  oscillator.frequency.value = BEEP.hz;
  gain.gain.value = BEEP.volume;
  oscillator.connect(gain).connect(audio.destination);
  oscillator.start();
  oscillator.stop(audio.currentTime + BEEP.seconds);
}
