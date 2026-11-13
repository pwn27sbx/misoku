package i18n

// Sync messages of the app.
const (
	SyncReadingStatus Key = "sync.reading_status"
	// The notes a sync brought, long (with where they came from) and
	// compact; plurals on the first count.
	SyncArrivedNewAndUpdated        Key = "sync.arrived.new_and_updated"
	SyncArrivedNew                  Key = "sync.arrived.new"
	SyncArrivedUpdated              Key = "sync.arrived.updated"
	SyncArrivedNewAndUpdatedCompact Key = "sync.arrived.new_and_updated.compact"
	SyncArrivedNewCompact           Key = "sync.arrived.new.compact"
	SyncArrivedUpdatedCompact       Key = "sync.arrived.updated.compact"
)

func init() {
	register(EN, map[Key]string{
		SyncReadingStatus: "Reading the sync status…",

		SyncArrivedNewAndUpdated + one:          "%d new note, %d updated from another computer",
		SyncArrivedNewAndUpdated + other:        "%d new notes, %d updated from another computer",
		SyncArrivedNew + one:                    "%d new note from another computer",
		SyncArrivedNew + other:                  "%d new notes from another computer",
		SyncArrivedUpdated + one:                "%d note updated from another computer",
		SyncArrivedUpdated + other:              "%d notes updated from another computer",
		SyncArrivedNewAndUpdatedCompact + one:   "%d new note, %d updated",
		SyncArrivedNewAndUpdatedCompact + other: "%d new notes, %d updated",
		SyncArrivedNewCompact + one:             "%d new note",
		SyncArrivedNewCompact + other:           "%d new notes",
		SyncArrivedUpdatedCompact + one:         "%d note updated",
		SyncArrivedUpdatedCompact + other:       "%d notes updated",
	})
	register(ES, map[Key]string{
		SyncReadingStatus: "Leyendo el estado de la sincronización…",

		SyncArrivedNewAndUpdated + one:          "%d nota nueva y %d con cambios desde otro equipo",
		SyncArrivedNewAndUpdated + other:        "%d notas nuevas y %d con cambios desde otro equipo",
		SyncArrivedNew + one:                    "%d nota nueva desde otro equipo",
		SyncArrivedNew + other:                  "%d notas nuevas desde otro equipo",
		SyncArrivedUpdated + one:                "%d nota actualizada desde otro equipo",
		SyncArrivedUpdated + other:              "%d notas actualizadas desde otro equipo",
		SyncArrivedNewAndUpdatedCompact + one:   "%d nota nueva y %d con cambios",
		SyncArrivedNewAndUpdatedCompact + other: "%d notas nuevas y %d con cambios",
		SyncArrivedNewCompact + one:             "%d nota nueva",
		SyncArrivedNewCompact + other:           "%d notas nuevas",
		SyncArrivedUpdatedCompact + one:         "%d nota actualizada",
		SyncArrivedUpdatedCompact + other:       "%d notas actualizadas",
	})
}
