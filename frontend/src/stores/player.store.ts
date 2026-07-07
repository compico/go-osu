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
    const isHydrated = ref(false);

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
        currentTime.value = audioService.currentTime;
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

    // ── Умный next/prev: пропускаем соседей с тем же groupKey ──
    function findAdjacent(direction: 1 | -1): Track | null {
        const list = playlist.value;
        if (!list.length || currentIndex.value === -1) return null;

        const cur = list[currentIndex.value];
        if (!cur.groupKey) {
            // нет группировки — обычный цикл по массиву
            const idx = (currentIndex.value + direction + list.length) % list.length;
            return list[idx];
        }

        for (
            let i = currentIndex.value + direction;
            i >= 0 && i < list.length;
            i += direction
        ) {
            if (list[i].groupKey !== cur.groupKey) return list[i];
        }
        return null;
    }

    const next = async (): Promise<void> => {
        const target = findAdjacent(1);
        if (target) await play(target);
    };

    const previous = async (): Promise<void> => {
        const target = findAdjacent(-1);
        if (target) await play(target);
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

    /**
     * Вызывается один раз при старте приложения (после restore из persist).
     * Подгружает аудио для восстановленного трека БЕЗ автоплея —
     * браузер всё равно не даст play() без жеста пользователя,
     * и это ожидаемо: трек просто готов, стоит на паузе.
     */
    const hydrate = async (): Promise<void> => {
        if (isHydrated.value) return;
        isHydrated.value = true;

        if (!currentTrack.value) return;

        try {
            isLoading.value = true;
            const restoredTime = currentTime.value;
            await audioService.load(currentTrack.value);
            duration.value = audioService.duration;
            if (restoredTime > 0) {
                audioService.seek(restoredTime);
                currentTime.value = restoredTime;
            }
            audioService.setVolume(volume.value);
            audioService.setMuted(isMuted.value);
            isPlaying.value = false; // явно: не играем сами по себе после reload
        } catch (err) {
            error.value = err instanceof Error ? err.message : 'Restore error';
        } finally {
            isLoading.value = false;
        }
    };

    // Event listeners
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
        hydrate,
    };
}, {
    persist: {
        key: 'osu-player',
        storage: localStorage,
        // playlist сюда НЕ кладём — он пересобирается на клике,
        // тащить весь текущий список карт в localStorage при каждом
        // изменении поиска — дорого и не нужно.
        pick: ['currentTrack', 'volume', 'isMuted', 'currentTime'],
    },
});
