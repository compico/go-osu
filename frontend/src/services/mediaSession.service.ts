export interface TrackMetadata {
    title: string
    artist: string
    album?: string
    artwork?: string
    duration?: number
}

export interface MediaSessionActions {
    play: () => void | Promise<void>
    pause: () => void
    nexttrack: () => void | Promise<void>
    previoustrack: () => void | Promise<void>
    seekto?: (time: number) => void
}

export class MediaSessionService {
    private audioElement: HTMLAudioElement | null = null
    private isInitialized = false

    constructor() {
        // Проверяем поддержку MediaSession API
        if (!('mediaSession' in navigator)) {
            console.warn('MediaSession API not supported')
            return
        }
        this.isInitialized = true
    }

    /**
     * Инициализирует audio element и привязывает события
     */
    initAudio(audio: HTMLAudioElement): void {
        this.audioElement = audio

        // Обновляем position state при воспроизведении
        audio.addEventListener('timeupdate', () => {
            this.updatePositionState()
        })

        audio.addEventListener('play', () => {
            this.setPlaybackState('playing')
        })

        audio.addEventListener('pause', () => {
            this.setPlaybackState('paused')
        })

        audio.addEventListener('ended', () => {
            this.setPlaybackState('none')
        })
    }

    /**
     * Устанавливает метаданные трека
     */
    setMetadata(metadata: TrackMetadata): void {
        if (!this.isInitialized) return

        const artwork = metadata.artwork ? [
            { src: metadata.artwork, sizes: '512x512', type: 'image/jpeg' }
        ] : []

        navigator.mediaSession.metadata = new MediaMetadata({
            title: metadata.title,
            artist: metadata.artist,
            album: metadata.album || '',
            artwork
        })
    }

    /**
     * Устанавливает действия для медиа-клавиш
     */
    setActionHandlers(actions: MediaSessionActions): void {
        if (!this.isInitialized) return

        navigator.mediaSession.setActionHandler('play', actions.play)
        navigator.mediaSession.setActionHandler('pause', actions.pause)
        navigator.mediaSession.setActionHandler('nexttrack', actions.nexttrack)
        navigator.mediaSession.setActionHandler('previoustrack', actions.previoustrack)

        if (actions.seekto) {
            navigator.mediaSession.setActionHandler('seekto', (details) => {
                if (details.seekTime !== undefined) {
                    actions.seekto!(details.seekTime)
                }
            })
        }

        // Дополнительные действия (опционально)
        navigator.mediaSession.setActionHandler('stop', actions.pause)
        navigator.mediaSession.setActionHandler('seekbackward', (details) => {
            if (this.audioElement && details.seekOffset) {
                this.audioElement.currentTime = Math.max(0, this.audioElement.currentTime - details.seekOffset)
            }
        })
        navigator.mediaSession.setActionHandler('seekforward', (details) => {
            if (this.audioElement && details.seekOffset) {
                this.audioElement.currentTime = Math.min(
                    this.audioElement.duration,
                    this.audioElement.currentTime + details.seekOffset
                )
            }
        })
    }

    /**
     * Устанавливает состояние воспроизведения
     */
    setPlaybackState(state: 'playing' | 'paused' | 'none'): void {
        if (!this.isInitialized) return
        navigator.mediaSession.playbackState = state
    }

    /**
     * Обновляет позицию воспроизведения
     */
    updatePositionState(): void {
        if (!this.isInitialized || !this.audioElement) return

        try {
            navigator.mediaSession.setPositionState({
                duration: this.audioElement.duration || 0,
                position: this.audioElement.currentTime,
                playbackRate: this.audioElement.playbackRate
            })
        } catch (e) {
            // Игнорируем ошибки если duration еще не загружен
        }
    }

    clearMetadata(): void {
        if (!this.isInitialized) return
        navigator.mediaSession.metadata = null
        navigator.mediaSession.playbackState = 'none'
    }
}

// Singleton instance
export const mediaSessionService = new MediaSessionService()