import { watch, type Ref } from 'vue'
import { mediaSessionService } from '@/services/mediaSession.service'

export function useMediaSession(
    audioRef: Ref<HTMLAudioElement | null>,
    playing: Ref<boolean>,
    selectedDiff: Ref<any>,
    selectedGroup: Ref<any>,
    playNext: () => void,
    playPrev: () => void,
    togglePlay: () => void
) {
    // Инициализация audio element при его появлении
    watch(audioRef, (audio) => {
        if (audio) {
            console.log('[MediaSession] Audio element initialized')
            mediaSessionService.initAudio(audio)

            mediaSessionService.setActionHandlers({
                play: () => {
                    if (!playing.value) togglePlay()
                },
                pause: () => {
                    if (playing.value) togglePlay()
                },
                nexttrack: playNext,
                previoustrack: playPrev,
                seekto: (time: number) => {
                    if (audioRef.value) {
                        audioRef.value.currentTime = time
                    }
                }
            })
        }
    }, { immediate: true })

    // Обновление метаданных при смене трека
    watch([selectedGroup, selectedDiff], ([group, diff]) => {
        if (group && diff) {
            mediaSessionService.setMetadata({
                title: group.song_name,
                artist: group.artist_name,
                album: diff.difficulty,
                artwork: `/api/osu/bg/${group.beatmap_id}.jpg`
            })
        } else {
            mediaSessionService.clearMetadata()
        }
    }, { immediate: true })

    // Обновление playback state
    watch(playing, (isPlaying) => {
        mediaSessionService.setPlaybackState(isPlaying ? 'playing' : 'paused')
    })
}