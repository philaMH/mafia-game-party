import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

class FakeUtterance {
  text: string;
  lang = "ko-KR";
  pitch = 1;
  rate = 1;
  volume = 1;
  voice: SpeechSynthesisVoice | null = null;
  onend: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(text: string) {
    this.text = text;
  }
}

class FakeSpeechSynthesis implements EventTarget {
  speaking = false;
  paused = false;
  pending = false;
  onvoiceschanged: ((this: SpeechSynthesis, ev: Event) => void) | null = null;
  utterances: FakeUtterance[] = [];

  speak(utt: SpeechSynthesisUtterance): void {
    const fake = utt as unknown as FakeUtterance;
    this.utterances.push(fake);
    this.speaking = true;
    // Synchronously fire onend so tests can observe queue progression.
    setTimeout(() => {
      this.speaking = false;
      fake.onend?.();
    }, 0);
  }
  cancel(): void {
    this.utterances = [];
    this.speaking = false;
  }
  pause(): void {
    this.paused = true;
  }
  resume(): void {
    this.paused = false;
  }
  getVoices(): SpeechSynthesisVoice[] {
    return [
      {
        default: true,
        lang: "ko-KR",
        localService: true,
        name: "FakeKorean",
        voiceURI: "fake-ko",
      } as SpeechSynthesisVoice,
    ];
  }
  addEventListener(): void {
    /* no-op for tests */
  }
  removeEventListener(): void {
    /* no-op for tests */
  }
  dispatchEvent(): boolean {
    return true;
  }
}

beforeEach(() => {
  Object.defineProperty(window, "speechSynthesis", {
    configurable: true,
    writable: true,
    value: new FakeSpeechSynthesis() as unknown as SpeechSynthesis,
  });
  Object.defineProperty(window, "SpeechSynthesisUtterance", {
    configurable: true,
    writable: true,
    value: FakeUtterance as unknown as typeof SpeechSynthesisUtterance,
  });
  vi.stubGlobal("WebSocket", class FakeWS {
    static OPEN = 1;
    readyState = 0;
    onopen: (() => void) | null = null;
    onmessage: ((ev: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    onerror: (() => void) | null = null;
    constructor(public url: string) {}
    send(_: string): void {}
    close(): void {
      this.readyState = 3;
      this.onclose?.();
    }
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  localStorage.clear();
});
