import { watch, onMounted } from 'vue'
import { mediaSessionService } from '@/services/mediaSession.service'
import { audioService } from '@/services/audio.service'
import { usePlayerStore } from '@/stores/player.store'

export function useMediaSession() {
    const player = usePlayerStore()

    onMounted(() => {
        mediaSessionService.initAudio(audioService.element)

        mediaSessionService.setActionHandlers({
            play: () => player.toggle(),
            pause: () => player.toggle(),
            nexttrack: () => player.next(),
            previoustrack: () => player.previous(),
            seekto: (time: number) => player.seek(time),
        })
    })

    watch(
        () => player.currentTrack,
        (track) => {
            if (track) {
                mediaSessionService.setMetadata({
                    title: track.title,
                    artist: track.artist ?? '',
                    album: track.album ?? '',
                    artwork: track.coverUrl,
                })
            } else {
                mediaSessionService.clearMetadata()
            }
        },
        { immediate: true },
    )

    watch(
        () => player.isPlaying,
        (isPlaying) => {
            mediaSessionService.setPlaybackState(isPlaying ? 'playing' : 'paused')
        },
    )
}
