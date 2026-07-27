package frontend

import "embed"

// Dist — собранные Vite-ассеты (npm run build).
// В репозитории обязательно должен лежать хотя бы один файл
// в dist/, иначе go:embed не скомпилируется на чистом клоне/CI.
//
//go:embed all:dist
var Dist embed.FS
