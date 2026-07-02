// stores/player.store.ts
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Track } from '@/types/player';
import { audioService } from '@/services/audio.service';

export const usePlayerStore = defineStore('player', () => {
    // State
    const currentTrack = ref<Track | null>(null);
    const playlist = ref<Track[]>([]);
    const currentTime = ref(0);
    const duration = ref(0);
    const volume = ref(1);
    const isMuted = ref(false);
    const isLoading = ref(false);
    const error = ref<string | null>(null);
    const isPlaying = ref(false);

    // Computed
    const isPaused = computed(() => !isPlaying.value && currentTrack.value !== null);
    const currentIndex = computed(() => {
        if (!currentTrack.value) return -1;
        return playlist.value.findIndex(t => t.id === currentTrack.value?.id);
    });

    // Actions
    const play = async (track?: Track): Promise<void> => {
        try {
            error.value = null;

            if (track) {
                if (currentTrack.value?.id === track.id && isPaused.value) {
                    await audioService.play();
                    isPlaying.value = true;
                    return;
                }

                isLoading.value = true;
                currentTrack.value = track;
                await audioService.load(track);
                duration.value = audioService.duration;
                isLoading.value = false;
            }

            await audioService.play();
            isPlaying.value = true;
        } catch (err) {
            error.value = err instanceof Error ? err.message : 'Playback error';
            isPlaying.value = false;
            isLoading.value = false;
        }
    };

    const pause = (): void => {
        audioService.pause();
        isPlaying.value = false;
    };

    const resume = async (): Promise<void> => {
        await play();
    };

    const stop = (): void => {
        audioService.stop();
        isPlaying.value = false;
        currentTime.value = 0;
    };

    const toggle = async (): Promise<void> => {
        if (isPlaying.value) {
            pause();
        } else {
            await resume();
        }
    };

    const next = async (): Promise<void> => {
        if (currentIndex.value === -1 || playlist.value.length === 0) return;

        const nextIndex = (currentIndex.value + 1) % playlist.value.length;
        await play(playlist.value[nextIndex]);
    };

    const previous = async (): Promise<void> => {
        if (currentIndex.value === -1 || playlist.value.length === 0) return;

        const prevIndex = currentIndex.value === 0
            ? playlist.value.length - 1
            : currentIndex.value - 1;
        await play(playlist.value[prevIndex]);
    };

    const seek = (time: number): void => {
        audioService.seek(time);
        currentTime.value = time;
    };

    const setPlaylist = (tracks: Track[]): void => {
        playlist.value = tracks;
    };

    const addToPlaylist = (track: Track): void => {
        if (!playlist.value.find(t => t.id === track.id)) {
            playlist.value.push(track);
        }
    };

    const removeFromPlaylist = (trackId: string): void => {
        playlist.value = playlist.value.filter(t => t.id !== trackId);
        if (currentTrack.value?.id === trackId) {
            stop();
            currentTrack.value = null;
        }
    };

    const clearPlaylist = (): void => {
        playlist.value = [];
        stop();
        currentTrack.value = null;
    };

    const setVolume = (newVolume: number): void => {
        const clampedVolume = Math.max(0, Math.min(1, newVolume));
        volume.value = clampedVolume;
        audioService.setVolume(clampedVolume);
        if (clampedVolume > 0 && isMuted.value) {
            isMuted.value = false;
            audioService.setMuted(false);
        }
    };

    const toggleMute = (): void => {
        isMuted.value = !isMuted.value;
        audioService.setMuted(isMuted.value);
    };

    // Setup event listeners
    audioService.onTimeUpdate((time) => {
        currentTime.value = time;
    });

    audioService.onEnded(() => {
        isPlaying.value = false;
        next();
    });

    audioService.onError((errorMessage) => {
        error.value = errorMessage;
        isPlaying.value = false;
        isLoading.value = false;
    });

    return {
        currentTrack,
        playlist,
        currentTime,
        duration,
        volume,
        isMuted,
        isLoading,
        error,
        isPlaying,
        isPaused,
        currentIndex,

        // Actions
        play,
        pause,
        resume,
        stop,
        toggle,
        next,
        previous,
        seek,
        setPlaylist,
        addToPlaylist,
        removeFromPlaylist,
        clearPlaylist,
        setVolume,
        toggleMute,
    };
});
