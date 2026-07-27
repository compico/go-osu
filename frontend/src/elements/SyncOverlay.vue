<script setup lang="ts">
import { computed } from 'vue'
import { useSyncProgress } from '@/lib/realtime'

const { history, latest, byKey, calcPercent, finished, connected } = useSyncProgress()

// Синк считается активным, если в этом сеансе уже пришло хоть одно событие
// и последний stage — не "done". Работает и если вкладка открылась посреди
// уже идущего синка (первое полученное событие не обязано быть read_db).
const active = computed(() => history.value.length > 0 && !finished.value)

// Главный прогресс-бар: приоритет — агрегат по битмапам (самая долгая фаза
// синка — расчёт сложности), затем текущая фаза записи в БД, иначе —
// неопределённый (полосатый/анимированный, т.к. под read_db/diff считать
// нечего).
const mainProgress = computed(() => {
  const calc = byKey['calc_progress']
  if (calc && calc.total > 0) {
    return { done: calc.done, total: calc.total, percent: calcPercent.value ?? 0 }
  }
  if (latest.value?.stage === 'write' && latest.value.total > 0) {
    const ev = latest.value
    return { done: ev.done, total: ev.total, percent: Math.round((ev.done / ev.total) * 100) }
  }
  return null
})

// Пока идёт расчёт сложности (самая долгая фаза), держим лейбл на ней,
// даже если параллельно долетают события записи skill_cache в БД —
// иначе лейбл дёргается между "Расчёт сложности" и "Запись в базу"
// каждые ~500мс, что выглядит как глюк, а не прогресс.
const stageLabel = computed(() => {
  if (finished.value) return 'Готово'

  const calc = byKey['calc_progress']
  if (calc && calc.done < calc.total) return 'Расчёт сложности'

  const ev = latest.value
  if (!ev) return 'Подключение…'

  switch (ev.stage) {
    case 'read_db':
      return 'Чтение osu!.db'
    case 'diff':
      return 'Сравнение с базой'
    case 'write':
      return 'Запись в базу'
    default:
      return ev.stage
  }
})

const errorEvents = computed(() => history.value.filter((ev) => ev.error))
const recentErrors = computed(() => errorEvents.value.slice(-5).reverse())
</script>

<template>
  <Transition name="fade">
    <div
        v-if="active"
        class="sync-overlay position-fixed top-0 start-0 w-100 h-100 d-flex align-items-center justify-content-center px-3"
    >
      <div class="card shadow-lg" style="max-width: 480px; width: 100%">
        <div class="card-body text-center p-4">
          <div class="spinner-border text-primary mb-3" role="status">
            <span class="visually-hidden">Синхронизация…</span>
          </div>

          <h5 class="card-title mb-1">Синхронизация библиотеки</h5>
          <p class="text-body-secondary mb-3">{{ stageLabel }}</p>

          <div v-if="mainProgress" class="mb-1">
            <div
                class="progress"
                role="progressbar"
                :aria-valuenow="mainProgress.percent"
                aria-valuemin="0"
                aria-valuemax="100"
            >
              <div class="progress-bar" :style="{ width: mainProgress.percent + '%' }">
                {{ mainProgress.percent }}%
              </div>
            </div>
            <div class="form-text mt-1 mb-0">{{ mainProgress.done }} / {{ mainProgress.total }} карт</div>
          </div>
          <div v-else class="progress mb-1" role="progressbar" aria-label="Синхронизация">
            <div class="progress-bar progress-bar-striped progress-bar-animated w-100"></div>
          </div>

          <div v-if="!connected" class="alert alert-warning small mt-3 mb-0 py-2">
            Соединение с сервером потеряно — ждём переподключения…
          </div>

          <div v-if="recentErrors.length" class="mt-3 text-start">
            <details>
              <summary class="small text-danger" role="button">
                Ошибок при синхронизации: {{ errorEvents.length }}
              </summary>
              <ul class="list-unstyled small text-body-secondary mt-2 mb-0">
                <li v-for="(ev, i) in recentErrors" :key="i" class="text-truncate">{{ ev.error }}</li>
              </ul>
            </details>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.sync-overlay {
  /* выше bootstrap-модалок (1055) и офканваса (1045), чтобы точно всё перекрыть */
  z-index: 1080;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>