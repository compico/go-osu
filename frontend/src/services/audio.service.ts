import type { Track } from '@/types/player';

export class AudioService {
    private audio: HTMLAudioElement;
    private timeUpdateCallbacks: Set<(time: number) => void> = new Set();
    private endedCallbacks: Set<() => void> = new Set();
    private errorCallbacks: Set<(error: string) => void> = new Set();

    constructor() {
        this.audio = new Audio();
        this.setupEventListeners();
    }

    get element(): HTMLAudioElement {
        return this.audio;
    }

    private setupEventListeners(): void {
        this.audio.addEventListener('timeupdate', () => {
            const time = this.audio.currentTime;
            this.timeUpdateCallbacks.forEach(cb => cb(time));
        });

        this.audio.addEventListener('ended', () => {
            this.endedCallbacks.forEach(cb => cb());
        });

        this.audio.addEventListener('error', () => {
            const error = this.audio.error?.message || 'Unknown audio error';
            this.errorCallbacks.forEach(cb => cb(error));
        });

        this.audio.addEventListener('loadedmetadata', () => {
            // Можно добавить callback для получения duration
        });
    }

    async load(track: Track): Promise<void> {
        return new Promise((resolve, reject) => {
            this.audio.src = track.url;
            this.audio.load();

            const onCanPlay = () => {
                this.audio.removeEventListener('canplay', onCanPlay);
                this.audio.removeEventListener('error', onError);
                resolve();
            };

            const onError = () => {
                this.audio.removeEventListener('canplay', onCanPlay);
                this.audio.removeEventListener('error', onError);
                reject(new Error('Failed to load audio'));
            };

            this.audio.addEventListener('canplay', onCanPlay);
            this.audio.addEventListener('error', onError);
        });
    }

    async play(): Promise<void> {
        try {
            await this.audio.play();
        } catch (error) {
            throw new Error('Playback failed');
        }
    }

    pause(): void {
        this.audio.pause();
    }

    stop(): void {
        this.audio.pause();
        this.audio.currentTime = 0;
    }

    seek(time: number): void {
        this.audio.currentTime = time;
    }

    setVolume(volume: number): void {
        this.audio.volume = Math.max(0, Math.min(1, volume));
    }

    setMuted(muted: boolean): void {
        this.audio.muted = muted;
    }

    get currentTime(): number {
        return this.audio.currentTime;
    }

    get duration(): number {
        return this.audio.duration || 0;
    }

    get volume(): number {
        return this.audio.volume;
    }

    get isMuted(): boolean {
        return this.audio.muted;
    }

    get isPlaying(): boolean {
        return !this.audio.paused && !this.audio.ended;
    }

    get isPaused(): boolean {
        return this.audio.paused;
    }

    // Подписка на события
    onTimeUpdate(callback: (time: number) => void): () => void {
        this.timeUpdateCallbacks.add(callback);
        return () => this.timeUpdateCallbacks.delete(callback);
    }

    onEnded(callback: () => void): () => void {
        this.endedCallbacks.add(callback);
        return () => this.endedCallbacks.delete(callback);
    }

    onError(callback: (error: string) => void): () => void {
        this.errorCallbacks.add(callback);
        return () => this.errorCallbacks.delete(callback);
    }

    destroy(): void {
        this.audio.pause();
        this.audio.src = '';
        this.timeUpdateCallbacks.clear();
        this.endedCallbacks.clear();
        this.errorCallbacks.clear();
    }
}

export const audioService = new AudioService();
