/**
 * Тэйпер громкости (по аналогии с "audio/log taper" потенциометрами).
 *
 * Слух воспринимает громкость логарифмически: линейный ползунок 0..1,
 * поданный напрямую в audio.volume, даёт резкий скачок в начале шкалы —
 * на 5% уже "громко". Степенная кривая (position^GAMMA) сжимает
 * чувствительность внизу шкалы и растягивает её ближе к максимуму.
 *
 * GAMMA зафиксирован на будущее — здесь можно добавить выбор кривой
 * пользователем (linear / gamma=2 / gamma=3 / gamma=4 и т.д.)
 */
const GAMMA = 3;

/** Позиция ползунка (0..1) -> реальная громкость, уходящая в audio.volume (0..1) */
export function sliderToVolume(sliderPos: number): number {
    const clamped = Math.max(0, Math.min(1, sliderPos));
    return Math.pow(clamped, GAMMA);
}

/** Реальная громкость (0..1) -> позиция ползунка (0..1), обратная функция */
export function volumeToSlider(volume: number): number {
    const clamped = Math.max(0, Math.min(1, volume));
    return Math.pow(clamped, 1 / GAMMA);
}