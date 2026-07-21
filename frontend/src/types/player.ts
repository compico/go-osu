import type { Ref, ComputedRef } from 'vue';

export interface Track {
    id: string;
    title: string;
    artist?: string;
    album?: string;
    duration?: number;
    url: string;
    coverUrl?: string;
    groupKey?: string;
    /** beatmap_set_id — используется для сравнения групп и для queue/adjacent на бэкенде */
    groupSetId?: number;
}

export interface PlayerStateRefs {
    currentTrack: Ref<Track | null>;
    playlist: Ref<Track[]>;
    currentTime: Ref<number>;
    duration: Ref<number>;
    volume: Ref<number>;
    isMuted: Ref<boolean>;
    isLoading: Ref<boolean>;
    error: Ref<string | null>;
    isPlaying: Ref<boolean>;
    isPaused: ComputedRef<boolean>;
    currentIndex: ComputedRef<number>;
}

export interface PlayerActions {
    play(track?: Track): Promise<void>;
    pause(): void;
    resume(): Promise<void>;
    stop(): void;
    toggle(): Promise<void>;
    next(): Promise<void>;
    previous(): Promise<void>;
    seek(time: number): void;
    setPlaylist(tracks: Track[]): void;
    addToPlaylist(track: Track): void;
    removeFromPlaylist(trackId: string): void;
    clearPlaylist(): void;
    setVolume(volume: number): void;
    toggleMute(): void;
}

export interface PlayerState {
    currentTrack: Track | null;
    playlist: Track[];
    currentTime: number;
    duration: number;
    volume: number;
    isMuted: boolean;
    isLoading: boolean;
    error: string | null;
    isPlaying: boolean;
    isPaused: boolean;
    currentIndex: number;
}