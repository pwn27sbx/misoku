package i18n

// The app's own words (internal/app): the bar's messages, its dialogs,
// menus and prompts. Keys start with "app." so they never meet another
// area's; a sentence that is built from parts in English is one entry per
// language here, so each language orders its own words.

// Sync, in the bar, the indicator and its dialogs.
const (
	AppSyncOffNotListed     Key = "app.sync.off_not_listed"
	AppSyncOffDisabled      Key = "app.sync.off_disabled"
	AppSyncOffConfig        Key = "app.sync.off_config"
	AppSyncOffWhy           Key = "app.sync.off_why"
	AppSyncing              Key = "app.sync.syncing"
	AppSyncAlready          Key = "app.sync.already"
	AppSyncPausedWhy        Key = "app.sync.paused_why"
	AppSyncPausedWhyKey     Key = "app.sync.paused_why_key"
	AppSyncFailed           Key = "app.sync.failed"
	AppSyncBusy             Key = "app.sync.busy"
	AppSyncErrorWhy         Key = "app.sync.error_why"
	AppSyncErrorWhyKey      Key = "app.sync.error_why_key"
	AppSynced               Key = "app.sync.synced"
	AppSyncConflictCopies   Key = "app.sync.conflict_copies"
	AppSyncFilesOdd         Key = "app.sync.files_odd"
	AppSyncOffline          Key = "app.sync.offline"
	AppSyncRetryWhy         Key = "app.sync.retry_why"
	AppSyncDeniedSignIn     Key = "app.sync.denied_sign_in"
	AppSyncMore             Key = "app.sync.more"
	AppSyncTooBig           Key = "app.sync.too_big"
	AppSyncStateSyncing     Key = "app.sync.state.syncing"
	AppSyncStateLock        Key = "app.sync.state.lock"
	AppSyncStateBusy        Key = "app.sync.state.busy"
	AppSyncStatePaused      Key = "app.sync.state.paused"
	AppSyncStateError       Key = "app.sync.state.error"
	AppSyncStateRetry       Key = "app.sync.state.retry"
	AppSyncStateOffline     Key = "app.sync.state.offline"
	AppSyncStateConflicts   Key = "app.sync.state.conflicts"
	AppSyncStateSynced      Key = "app.sync.state.synced"
	AppSyncStateFiles       Key = "app.sync.state.files"
	AppSyncEveryOff         Key = "app.sync.every_off"
	AppSyncEvery            Key = "app.sync.every"
	AppAgoNow               Key = "app.ago.now"
	AppAgoMinutes           Key = "app.ago.minutes"
	AppAgoHours             Key = "app.ago.hours"
	AppAgoDays              Key = "app.ago.days"
	AppSyncLocked           Key = "app.sync.locked"
	AppSyncLockLine1        Key = "app.sync.lock_line1"
	AppSyncLockLine2        Key = "app.sync.lock_line2"
	AppSyncLockLine3        Key = "app.sync.lock_line3"
	AppSyncStatusTitle      Key = "app.sync.status_title"
	AppSyncRemoveLock       Key = "app.sync.remove_lock"
	AppSyncLockNotRemoved   Key = "app.sync.lock_not_removed"
	AppSyncPausedKey        Key = "app.sync.paused_key"
	AppSyncErrorKey         Key = "app.sync.error_key"
	AppSyncOfflineKey       Key = "app.sync.offline_key"
	AppSyncKeyRetries       Key = "app.sync.key_retries"
	AppSyncKeySignsIn       Key = "app.sync.key_signs_in"
	AppSyncKeyOpensIt       Key = "app.sync.key_opens_it"
	AppSyncKeySpace         Key = "app.sync.key_space"
	AppSyncRewrittenBar     Key = "app.sync.rewritten.bar"
	AppSyncRewrittenWhy     Key = "app.sync.rewritten.why"
	AppSyncUnrelatedBar     Key = "app.sync.unrelated.bar"
	AppSyncUnrelatedWhy     Key = "app.sync.unrelated.why"
	AppSyncNoJoin           Key = "app.sync.no_join"
	AppSyncInProgressBar    Key = "app.sync.in_progress.bar"
	AppSyncInProgressWhy    Key = "app.sync.in_progress.why"
	AppSyncInProgressNo     Key = "app.sync.in_progress.no"
	AppSyncDetachedBar      Key = "app.sync.detached.bar"
	AppSyncDetachedWhy      Key = "app.sync.detached.why"
	AppSyncDetachedNo       Key = "app.sync.detached.no"
	AppSyncPausedTitle      Key = "app.sync.paused_title"
	AppSyncDeniedTitle      Key = "app.sync.denied_title"
	AppSyncDeniedWhy        Key = "app.sync.denied_why"
	AppSyncNothingChanged   Key = "app.sync.nothing_changed"
	AppSyncCheckAgainHint   Key = "app.sync.check_again_hint"
	AppSyncCheckAgain       Key = "app.sync.check_again"
	AppSyncSignInLine1      Key = "app.sync.sign_in.line1"
	AppSyncSignInLine2      Key = "app.sync.sign_in.line2"
	AppSyncSignInLine3      Key = "app.sync.sign_in.line3"
	AppSyncSignInTitle      Key = "app.sync.sign_in.title"
	AppSyncSignIn           Key = "app.sync.sign_in.button"
	AppSyncCheckingSignIn   Key = "app.sync.checking_sign_in"
	AppSyncStillPaused      Key = "app.sync.still_paused"
	AppSyncLockOpensIt      Key = "app.sync.lock_opens_it"
	AppSyncNotYet           Key = "app.sync.not_yet"
	AppSyncInfoState        Key = "app.sync.info.state"
	AppSyncInfoRemote       Key = "app.sync.info.remote"
	AppSyncInfoBranch       Key = "app.sync.info.branch"
	AppSyncInfoTimed        Key = "app.sync.info.timed"
	AppSyncInfoPending      Key = "app.sync.info.pending"
	AppSyncInfoLastOK       Key = "app.sync.info.last_ok"
	AppSyncInfoLastErr      Key = "app.sync.info.last_err"
	AppSyncInfoNoErr        Key = "app.sync.info.no_err"
	AppSyncInfoUnreadable   Key = "app.sync.info.unreadable"
	AppSyncInfoConfig       Key = "app.sync.info.config"
	AppSyncInfoNoConflicts  Key = "app.sync.info.no_conflicts"
	AppSyncInfoConflicts    Key = "app.sync.info.conflicts"
	AppSyncInfoFiles        Key = "app.sync.info.files"
	AppSyncInfoArrived      Key = "app.sync.info.arrived"
	AppSyncInfoStopped      Key = "app.sync.info.stopped"
	AppSyncInfoRemoveAgain  Key = "app.sync.info.remove_again"
	AppListMore             Key = "app.list_more"
	AppNone                 Key = "app.none"
	AppSyncPushing          Key = "app.sync.pushing"
	AppSyncNotPushedRunning Key = "app.sync.not_pushed_running"
	AppSyncNotPushedBusy    Key = "app.sync.not_pushed_busy"
	AppSyncNotPushedRetry   Key = "app.sync.not_pushed_retry"
	AppSyncNotPushedPaused  Key = "app.sync.not_pushed_paused"
	AppSyncPushed           Key = "app.sync.pushed"
	AppSyncNothingToPush    Key = "app.sync.nothing_to_push"
	AppSyncNoAnswer         Key = "app.sync.no_answer"
)

func init() {
	register(EN, map[Key]string{
		AppSyncOffNotListed: "Sync is off: this folder is not listed in sync.vaults",
		AppSyncOffDisabled:  "Sync is off (MISOKU_SYNC=0)",
		AppSyncOffConfig:    "Sync is off: the config file could not be read (%s)",
		AppSyncOffWhy:       "%s (%s)",
		AppSyncing:          "Syncing…",
		AppSyncAlready:      "Already syncing…",
		AppSyncPausedWhy:    "Sync paused: %s",
		AppSyncPausedWhyKey: "Sync paused: %s · %s",
		AppSyncFailed:       "Sync failed: %s",
		AppSyncBusy:         "Sync: another Misoku window syncs this vault",
		AppSyncErrorWhy:     "Sync error: %s",
		AppSyncErrorWhyKey:  "Sync error: %s · %s",
		AppSynced:           "Synced",

		AppSyncConflictCopies + one:   " · %d conflict copy: %s",
		AppSyncConflictCopies + other: " · %d conflict copies: %s",

		AppSyncFilesOdd:     " · %d files do not sync as they are · :sync status",
		AppSyncOffline:      "Sync: offline, %d pending · %s retries",
		AppSyncRetryWhy:     "Sync stopped, it tries again next time: %s",
		AppSyncDeniedSignIn: "Sync paused: GitHub did not let this computer in · %s signs in",
		AppSyncMore:         " ... +%d more · :sync status",
		AppSyncTooBig:       "%s is too big and stays out of sync",

		AppSyncStateSyncing: "󰓦 syncing…",
		AppSyncStateLock:    "󰓦 sync paused, lock left",
		AppSyncStateBusy:    "󰓦 other window syncs",
		AppSyncStatePaused:  "󰓦 sync paused",
		AppSyncStateError:   "󰓦 sync error",
		AppSyncStateRetry:   "󰓦 sync will retry",
		AppSyncStateOffline: "󰓦 offline · %d pending",

		AppSyncStateConflicts + one:   "󰓦 %d conflict",
		AppSyncStateConflicts + other: "󰓦 %d conflicts",
		AppSyncStateSynced:            "󰓦 synced %s",
		AppSyncStateFiles + one:       " · %d file to check",
		AppSyncStateFiles + other:     " · %d files to check",

		AppSyncEveryOff: "off",
		AppSyncEvery:    "every %s",
		AppAgoNow:       "just now",
		AppAgoMinutes:   "%dm ago",
		AppAgoHours:     "%dh ago",
		AppAgoDays:      "%dd ago",

		AppSyncLocked:         "a git lock file is in the way (.git/index.lock)",
		AppSyncLockLine1:      "A git lock file (.git/index.lock) is in this vault, so sync cannot go on.",
		AppSyncLockLine2:      "It may belong to a git command still running, for example a commit in a terminal:",
		AppSyncLockLine3:      "remove it only if no git command is running.",
		AppSyncStatusTitle:    "Sync status",
		AppSyncRemoveLock:     "Remove the lock and sync",
		AppSyncLockNotRemoved: "Lock not removed: %s",
		AppSyncPausedKey:      "Sync paused · %s",
		AppSyncErrorKey:       "Sync error · %s",
		AppSyncOfflineKey:     "Sync offline · %s",
		AppSyncKeyRetries:     "%s retries",
		AppSyncKeySignsIn:     "%s signs in",
		AppSyncKeyOpensIt:     "%s opens it",
		AppSyncKeySpace:       "space %s",

		AppSyncRewrittenBar:  "this vault and origin no longer match",
		AppSyncRewrittenWhy:  "The history in origin was rewritten, so this vault and origin no longer match.",
		AppSyncUnrelatedBar:  "this vault and origin share no history",
		AppSyncUnrelatedWhy:  "This vault and origin share no history: they look like two separate sets of notes.",
		AppSyncNoJoin:        "Joining them could bring back something deleted on purpose, so Misoku does not offer it.",
		AppSyncInProgressBar: "an unfinished merge or rebase is waiting in this vault",
		AppSyncInProgressWhy: "A merge, rebase or similar operation started in this vault is not finished.",
		AppSyncInProgressNo:  "Misoku does not finish or cancel it: it may hold choices only you can make.",
		AppSyncDetachedBar:   "this vault is not on a branch",
		AppSyncDetachedWhy:   "This vault is not on a branch right now, so what you write has nowhere to go in origin.",
		AppSyncDetachedNo:    "Misoku does not switch branches: the notes on your screen would change.",

		AppSyncPausedTitle:    "Sync is paused",
		AppSyncDeniedTitle:    "Sync cannot get in",
		AppSyncDeniedWhy:      "Origin did not let this computer in.",
		AppSyncNothingChanged: "Nothing was changed, and your notes are safe here.",
		AppSyncCheckAgainHint: "Once it is sorted out, choose Check again.",
		AppSyncCheckAgain:     "Check again",
		AppSyncSignInLine1:    "GitHub did not let this computer in, so sync stopped. Your notes are safe here.",
		AppSyncSignInLine2:    "Sign in: gh shows a one-time code and opens your browser. gh keeps the access; Misoku never sees it.",
		AppSyncSignInLine3:    "Sync runs once you are back.",
		AppSyncSignInTitle:    "Sign in to GitHub",
		AppSyncSignIn:         "Sign in",
		AppSyncCheckingSignIn: "Checking the sign-in…",
		AppSyncStillPaused:    "Sync still paused: %s · %s",
		AppSyncLockOpensIt:    "Sync paused: %s · %s opens it",

		AppSyncNotYet:        "not synced yet in this window",
		AppSyncInfoState:     "State: %s",
		AppSyncInfoRemote:    "Remote: %s",
		AppSyncInfoBranch:    "Branch: %s",
		AppSyncInfoTimed:     "Timed sync: %s",
		AppSyncInfoLastOK:    "Last synced: %s",
		AppSyncInfoLastErr:   "Last error (%s): %s",
		AppSyncInfoNoErr:     "Last error: none",
		AppSyncInfoConfig:    "Config: %s",
		AppSyncInfoArrived:   "%s (%s):",
		AppSyncInfoStopped:   "Sync stopped: %s.",
		AppListMore:          "  … and %d more",
		AppNone:              "(none)",
		AppSyncPushing:       "Pushing changes…",
		AppSyncPushed:        "Changes pushed.",
		AppSyncNothingToPush: "Nothing new to push.",
		AppSyncNoAnswer:      "no answer within %s",

		AppSyncInfoPending + one:   "Pending: %d commit to push",
		AppSyncInfoPending + other: "Pending: %d commits to push",

		AppSyncInfoUnreadable:   "Could not read: %s",
		AppSyncInfoNoConflicts:  "Conflict copies: none",
		AppSyncInfoConflicts:    "Conflict copies (%d), delete each once resolved:",
		AppSyncInfoFiles:        "Files that do not sync as they are (%d):",
		AppSyncInfoRemoveAgain:  "To remove the lock, run :sync status again.",
		AppSyncNotPushedRunning: "Not pushed, the next sync tries again: a sync still running did not finish (%s)",
		AppSyncNotPushedBusy:    "Not pushed: another Misoku window syncs this vault",
		AppSyncNotPushedRetry:   "Not pushed, the next sync tries again: %s",
		AppSyncNotPushedPaused:  "Not pushed, sync is paused: %s",
	})
	register(ES, map[Key]string{
		AppSyncOffNotListed: "La sincronización está desactivada: esta carpeta no figura en sync.vaults",
		AppSyncOffDisabled:  "La sincronización está desactivada (MISOKU_SYNC=0)",
		AppSyncOffConfig:    "La sincronización está desactivada: no se pudo leer el archivo de configuración (%s)",
		AppSyncOffWhy:       "%s (%s)",
		AppSyncing:          "Sincronizando…",
		AppSyncAlready:      "Ya se está sincronizando…",
		AppSyncPausedWhy:    "Sincronización en pausa: %s",
		AppSyncPausedWhyKey: "Sincronización en pausa: %s · %s",
		AppSyncFailed:       "La sincronización falló: %s",
		AppSyncBusy:         "Sincronización: otra ventana de Misoku sincroniza este vault",
		AppSyncErrorWhy:     "Error de sincronización: %s",
		AppSyncErrorWhyKey:  "Error de sincronización: %s · %s",
		AppSynced:           "Sincronizado",

		AppSyncConflictCopies + one:   " · %d copia en conflicto: %s",
		AppSyncConflictCopies + other: " · %d copias en conflicto: %s",

		AppSyncFilesOdd:     " · %d archivos no se sincronizan tal cual · :sync status",
		AppSyncOffline:      "Sincronización: sin conexión, %d pendientes · %s reintenta",
		AppSyncRetryWhy:     "La sincronización se detuvo, lo intentará de nuevo la próxima vez: %s",
		AppSyncDeniedSignIn: "Sincronización en pausa: GitHub no dejó entrar a este equipo · %s inicia sesión",
		AppSyncMore:         " ... +%d más · :sync status",
		AppSyncTooBig:       "%s es demasiado grande y queda fuera de la sincronización",

		AppSyncStateSyncing: "󰓦 sincronizando…",
		AppSyncStateLock:    "󰓦 sincronización en pausa, quedó un bloqueo",
		AppSyncStateBusy:    "󰓦 otra ventana sincroniza",
		AppSyncStatePaused:  "󰓦 sincronización en pausa",
		AppSyncStateError:   "󰓦 error de sincronización",
		AppSyncStateRetry:   "󰓦 se reintentará",
		AppSyncStateOffline: "󰓦 sin conexión · %d pendientes",

		AppSyncStateConflicts + one:   "󰓦 %d conflicto",
		AppSyncStateConflicts + other: "󰓦 %d conflictos",
		AppSyncStateSynced:            "󰓦 sincronizado %s",
		AppSyncStateFiles + one:       " · %d archivo por revisar",
		AppSyncStateFiles + other:     " · %d archivos por revisar",

		AppSyncEveryOff: "desactivada",
		AppSyncEvery:    "cada %s",
		AppAgoNow:       "hace un momento",
		AppAgoMinutes:   "hace %d min",
		AppAgoHours:     "hace %d h",
		AppAgoDays:      "hace %d d",

		AppSyncLocked:         "un archivo de bloqueo de git estorba (.git/index.lock)",
		AppSyncLockLine1:      "Hay un archivo de bloqueo de git (.git/index.lock) en este vault, así que la sincronización no puede seguir.",
		AppSyncLockLine2:      "Puede ser de un comando de git que sigue en marcha, por ejemplo un commit en una terminal:",
		AppSyncLockLine3:      "quítalo solo si no hay ningún comando de git en marcha.",
		AppSyncStatusTitle:    "Estado de la sincronización",
		AppSyncRemoveLock:     "Quitar el bloqueo y sincronizar",
		AppSyncLockNotRemoved: "No se quitó el bloqueo: %s",
		AppSyncPausedKey:      "Sincronización en pausa · %s",
		AppSyncErrorKey:       "Error de sincronización · %s",
		AppSyncOfflineKey:     "Sincronización sin conexión · %s",
		AppSyncKeyRetries:     "%s reintenta",
		AppSyncKeySignsIn:     "%s inicia sesión",
		AppSyncKeyOpensIt:     "%s lo abre",
		AppSyncKeySpace:       "espacio %s",

		AppSyncRewrittenBar:  "este vault y origin ya no coinciden",
		AppSyncRewrittenWhy:  "La historia de origin se reescribió, así que este vault y origin ya no coinciden.",
		AppSyncUnrelatedBar:  "este vault y origin no comparten historia",
		AppSyncUnrelatedWhy:  "Este vault y origin no comparten historia: parecen dos conjuntos de notas distintos.",
		AppSyncNoJoin:        "Unirlos podría traer de vuelta algo borrado a propósito, así que Misoku no lo ofrece.",
		AppSyncInProgressBar: "hay un merge o rebase sin terminar en este vault",
		AppSyncInProgressWhy: "Un merge, rebase u operación parecida que empezó en este vault no ha terminado.",
		AppSyncInProgressNo:  "Misoku no lo termina ni lo cancela: puede tener decisiones que solo tú puedes tomar.",
		AppSyncDetachedBar:   "este vault no está en una rama",
		AppSyncDetachedWhy:   "Este vault no está en una rama ahora mismo, así que lo que escribas no tiene adónde ir en origin.",
		AppSyncDetachedNo:    "Misoku no cambia de rama: cambiarían las notas que tienes en pantalla.",

		AppSyncPausedTitle:    "La sincronización está en pausa",
		AppSyncDeniedTitle:    "La sincronización no puede entrar",
		AppSyncDeniedWhy:      "Origin no dejó entrar a este equipo.",
		AppSyncNothingChanged: "No se cambió nada y tus notas están a salvo aquí.",
		AppSyncCheckAgainHint: "Cuando esté resuelto, elige Comprobar de nuevo.",
		AppSyncCheckAgain:     "Comprobar de nuevo",
		AppSyncSignInLine1:    "GitHub no dejó entrar a este equipo, así que la sincronización se detuvo. Tus notas están a salvo aquí.",
		AppSyncSignInLine2:    "Iniciar sesión: gh muestra un código de un solo uso y abre tu navegador. gh guarda el acceso; Misoku nunca lo ve.",
		AppSyncSignInLine3:    "La sincronización se ejecuta en cuanto vuelvas.",
		AppSyncSignInTitle:    "Iniciar sesión en GitHub",
		AppSyncSignIn:         "Iniciar sesión",
		AppSyncCheckingSignIn: "Comprobando el inicio de sesión…",
		AppSyncStillPaused:    "La sincronización sigue en pausa: %s · %s",
		AppSyncLockOpensIt:    "Sincronización en pausa: %s · %s lo abre",

		AppSyncNotYet:        "aún sin sincronizar en esta ventana",
		AppSyncInfoState:     "Estado: %s",
		AppSyncInfoRemote:    "Remoto: %s",
		AppSyncInfoBranch:    "Rama: %s",
		AppSyncInfoTimed:     "Sincronización periódica: %s",
		AppSyncInfoLastOK:    "Última sincronización: %s",
		AppSyncInfoLastErr:   "Último error (%s): %s",
		AppSyncInfoNoErr:     "Último error: ninguno",
		AppSyncInfoConfig:    "Configuración: %s",
		AppSyncInfoArrived:   "%s (%s):",
		AppSyncInfoStopped:   "Sincronización detenida: %s.",
		AppListMore:          "  … y %d más",
		AppNone:              "(ninguno)",
		AppSyncPushing:       "Enviando los cambios…",
		AppSyncPushed:        "Cambios enviados.",
		AppSyncNothingToPush: "No hay nada nuevo que enviar.",
		AppSyncNoAnswer:      "sin respuesta en %s",

		AppSyncInfoPending + one:   "Pendiente: %d commit por enviar",
		AppSyncInfoPending + other: "Pendientes: %d commits por enviar",

		AppSyncInfoUnreadable:   "No se pudo leer: %s",
		AppSyncInfoNoConflicts:  "Copias en conflicto: ninguna",
		AppSyncInfoConflicts:    "Copias en conflicto (%d), borra cada una cuando esté resuelta:",
		AppSyncInfoFiles:        "Archivos que no se sincronizan tal cual (%d):",
		AppSyncInfoRemoveAgain:  "Para quitar el bloqueo, ejecuta :sync status otra vez.",
		AppSyncNotPushedRunning: "No se envió, la próxima sincronización lo intentará de nuevo: una sincronización en marcha no terminó (%s)",
		AppSyncNotPushedBusy:    "No se envió: otra ventana de Misoku sincroniza este vault",
		AppSyncNotPushedRetry:   "No se envió, la próxima sincronización lo intentará de nuevo: %s",
		AppSyncNotPushedPaused:  "No se envió, la sincronización está en pausa: %s",
	})
}

// Words many screens share: buttons and short answers.
const (
	AppCancel        Key = "app.cancel"
	AppClose         Key = "app.close"
	AppYes           Key = "app.yes"
	AppNo            Key = "app.no"
	AppSave          Key = "app.save"
	AppDelete        Key = "app.delete"
	AppRename        Key = "app.rename"
	AppOpen          Key = "app.open"
	AppDiscard       Key = "app.discard"
	AppCancelledWord Key = "app.cancelled_word"
	AppNotFound      Key = "app.not_found"
	AppCannotOpen    Key = "app.cannot_open"
	AppNothingHere   Key = "app.nothing_matches"
)

func init() {
	register(EN, map[Key]string{
		AppCancel:        "Cancel",
		AppClose:         "Close",
		AppYes:           "Yes",
		AppNo:            "No",
		AppSave:          "Save",
		AppDelete:        "Delete",
		AppRename:        "Rename",
		AppOpen:          "Open",
		AppDiscard:       "Discard",
		AppCancelledWord: "cancelled",
		AppNotFound:      "Not found: %s",
		AppCannotOpen:    "Cannot open %s",
		AppNothingHere:   "   Nothing matches",
	})
	register(ES, map[Key]string{
		AppCancel:        "Cancelar",
		AppClose:         "Cerrar",
		AppYes:           "Sí",
		AppNo:            "No",
		AppSave:          "Guardar",
		AppDelete:        "Eliminar",
		AppRename:        "Renombrar",
		AppOpen:          "Abrir",
		AppDiscard:       "Descartar",
		AppCancelledWord: "cancelado",
		AppNotFound:      "No se encontró: %s",
		AppCannotOpen:    "No se puede abrir %s",
		AppNothingHere:   "   Nada coincide",
	})
}

// Folder edits applied from the explorer, link rewrites and mentions.
const (
	AppPlanCantApply     Key = "app.plan.cant_apply"
	AppPlanNoChanges     Key = "app.plan.no_changes"
	AppPlanMore          Key = "app.plan.more"
	AppPlanUpdates       Key = "app.plan.updates"
	AppPlanPermanent     Key = "app.plan.permanent"
	AppPlanTrash         Key = "app.plan.trash"
	AppPlanTitle         Key = "app.plan.title"
	AppPlanUnknown       Key = "app.plan.unknown"
	AppPlanApplied       Key = "app.plan.applied"
	AppPlanStopped       Key = "app.plan.stopped"
	AppLinksInNotes      Key = "app.links_in_notes"
	AppLinksInNotesLinks Key = "app.links_in_notes.links"
	AppLinksInNotesNotes Key = "app.links_in_notes.notes"
	AppRelinkTitle       Key = "app.relink.title"
	AppRelinkNo          Key = "app.relink.no"
	AppRelinkUpdated     Key = "app.relink.updated"
	AppRelinkSkipped     Key = "app.relink.skipped"
	AppMentionDirty      Key = "app.mention.dirty"
	AppMentionTitle      Key = "app.mention.title"
	AppMentionNo         Key = "app.mention.no"
	AppMentionCount      Key = "app.mention.count"
	AppMentionSkipped    Key = "app.mention.skipped"
	AppMentionLinked     Key = "app.mention.linked"
)

func init() {
	register(EN, map[Key]string{
		AppPlanCantApply: "Can't apply: %s",
		AppPlanNoChanges: "No changes",
		AppPlanMore:      "…and %d more",
		AppPlanUpdates:   "Updates %s.",
		AppPlanPermanent: "Deleted entries are removed from disk, cannot be undone.",
		AppPlanTrash:     "Deleted entries go to the Trash.",
		AppPlanTitle:     "CONFIRM FILE SYSTEM ACTIONS",
		AppPlanUnknown:   "unknown action %s",

		AppPlanApplied + one:   "Applied %d action",
		AppPlanApplied + other: "Applied %d actions",

		AppPlanStopped:  "Stopped after %d of %d actions: %v · not done: %s",
		AppLinksInNotes: "%s in %s",

		AppLinksInNotesLinks + one:   "%d link",
		AppLinksInNotesLinks + other: "%d links",
		AppLinksInNotesNotes + one:   "%d note",
		AppLinksInNotesNotes + other: "%d notes",

		AppRelinkTitle:    "Update %s?",
		AppRelinkNo:       "No keeps the new name and leaves the links as written.",
		AppRelinkUpdated:  "%s · updated %s",
		AppRelinkSkipped:  " · %s changed meanwhile, its links were left as written",
		AppMentionDirty:   "%s has unsaved changes: save it before linking its mentions",
		AppMentionTitle:   "Link %s in %s?",
		AppMentionNo:      "Each one becomes [[%s]]; No leaves the note as written.",
		AppMentionSkipped: "%s changed since it was read; its mentions were left as written",
		AppMentionLinked:  "Linked %s in %s",

		AppMentionCount + one:   "%d mention",
		AppMentionCount + other: "%d mentions",
	})
	register(ES, map[Key]string{
		AppPlanCantApply: "No se puede aplicar: %s",
		AppPlanNoChanges: "No hay cambios",
		AppPlanMore:      "…y %d más",
		AppPlanUpdates:   "Actualiza %s.",
		AppPlanPermanent: "Lo eliminado se borra del disco y no se puede deshacer.",
		AppPlanTrash:     "Lo eliminado va a la Papelera.",
		AppPlanTitle:     "CONFIRMAR CAMBIOS EN LOS ARCHIVOS",
		AppPlanUnknown:   "acción desconocida %s",

		AppPlanApplied + one:   "Se aplicó %d acción",
		AppPlanApplied + other: "Se aplicaron %d acciones",

		AppPlanStopped:  "Se detuvo tras %d de %d acciones: %v · sin hacer: %s",
		AppLinksInNotes: "%s en %s",

		AppLinksInNotesLinks + one:   "%d enlace",
		AppLinksInNotesLinks + other: "%d enlaces",
		AppLinksInNotesNotes + one:   "%d nota",
		AppLinksInNotesNotes + other: "%d notas",

		AppRelinkTitle:    "¿Actualizar %s?",
		AppRelinkNo:       "No mantiene el nombre nuevo y deja los enlaces como están.",
		AppRelinkUpdated:  "%s · se actualizaron %s",
		AppRelinkSkipped:  " · %s cambió mientras tanto, sus enlaces quedaron como estaban",
		AppMentionDirty:   "%s tiene cambios sin guardar: guárdala antes de enlazar sus menciones",
		AppMentionTitle:   "¿Enlazar %s en %s?",
		AppMentionNo:      "Cada una se convierte en [[%s]]; No deja la nota como está.",
		AppMentionSkipped: "%s cambió desde que se leyó; sus menciones quedaron como estaban",
		AppMentionLinked:  "Se enlazaron %s en %s",

		AppMentionCount + one:   "%d mención",
		AppMentionCount + other: "%d menciones",
	})
}

// Keys, saving, renaming, deleting and quitting; the screen's frame.
const (
	AppNotLeaderKey      Key = "app.not_leader_key"
	AppNeedsOpenNote     Key = "app.needs_open_note"
	AppNoNoteForLink     Key = "app.no_note_for_link"
	AppLearnNotSaved     Key = "app.learn_not_saved"
	AppFileDialogFailed  Key = "app.file_dialog_failed"
	AppNoFileName        Key = "app.no_file_name"
	AppRenameFailed      Key = "app.rename_failed"
	AppRenamedTo         Key = "app.renamed_to"
	AppDeleteFile        Key = "app.delete.file"
	AppDeleteFolder      Key = "app.delete.folder"
	AppDeleteNote        Key = "app.delete.note"
	AppDeleteFileGone    Key = "app.delete.file_gone"
	AppDeleteFolderGone  Key = "app.delete.folder_gone"
	AppDeleteNoteGone    Key = "app.delete.note_gone"
	AppDeleteRestore     Key = "app.delete.restore"
	AppDeleteForGood     Key = "app.delete.for_good"
	AppDeleteTitle       Key = "app.delete.title"
	AppDeleteFailed      Key = "app.delete.failed"
	AppDeleted           Key = "app.deleted"
	AppLoading           Key = "app.loading"
	AppTooSmall          Key = "app.too_small"
	AppTooNarrowPreview  Key = "app.too_narrow_preview"
	AppTabNew            Key = "app.tab_new"
	AppModeHome          Key = "app.mode.home"
	AppModeExplorer      Key = "app.mode.explorer"
	AppModePreview       Key = "app.mode.preview"
	AppCmdlineTitle      Key = "app.cmdline.title"
	AppSearchTitle       Key = "app.cmdline.search_title"
	AppWords             Key = "app.words"
	AppLnCol             Key = "app.ln_col"
	AppNoFile            Key = "app.no_file"
	AppUnsavedMark       Key = "app.unsaved_mark"
	AppTableHint         Key = "app.table_hint"
	AppErrorHint         Key = "app.error_hint"
	AppUntitled          Key = "app.untitled"
	AppUnsavedTitle      Key = "app.unsaved.title"
	AppUnsavedQuestion   Key = "app.unsaved.question"
	AppUnsavedLost       Key = "app.unsaved.lost"
	AppDontSave          Key = "app.dont_save"
	AppGoneTitle         Key = "app.gone.title"
	AppGoneWhat          Key = "app.gone.what"
	AppGoneAsk           Key = "app.gone.ask"
	AppRecreate          Key = "app.recreate"
	AppNotSavedKeep      Key = "app.not_saved_keep"
	AppBothKept          Key = "app.both_kept"
	AppSaveFailed        Key = "app.save_failed"
	AppWritten           Key = "app.written"
	AppGoneSaveAsks      Key = "app.gone_save_asks"
	AppQuitMore          Key = "app.quit.more"
	AppQuitTitle         Key = "app.quit.title"
	AppQuitUnsaved       Key = "app.quit.unsaved"
	AppSaveAll           Key = "app.save_all"
	AppChangedOnDiskKept Key = "app.changed_on_disk_kept"
	AppReloaded          Key = "app.reloaded"
)

func init() {
	register(EN, map[Key]string{
		AppNotLeaderKey:     "%s: not a leader key",
		AppNeedsOpenNote:    "%s: needs an open note",
		AppNoNoteForLink:    "No note found for this link",
		AppLearnNotSaved:    "Could not remember the answer (%s); Misoku will ask again next time",
		AppFileDialogFailed: "File dialog failed: %s",
		AppNoFileName:       "E32: No file name",
		AppRenameFailed:     "Rename failed: %s",
		AppRenamedTo:        "Renamed to %s",
		AppDeleteFile:       "The file is moved to the Trash.",
		AppDeleteFolder:     "The folder and everything inside it are moved to the Trash.",
		AppDeleteNote:       "The note and all of its sub-notes are moved to the Trash.",
		AppDeleteFileGone:   "The file is removed from disk.",
		AppDeleteFolderGone: "The folder and everything inside it are removed from disk.",
		AppDeleteNoteGone:   "The note and all of its sub-notes are removed from disk.",
		AppDeleteRestore:    "You can restore it from there.",
		AppDeleteForGood:    "This cannot be undone.",
		AppDeleteTitle:      "Delete %s?",
		AppDeleteFailed:     "Delete failed: %s",
		AppDeleted:          "Deleted %s",
		AppLoading:          "loading…",
		AppTooSmall:         "Terminal too small (%dx%d)\nResize to at least %dx%d",
		AppTooNarrowPreview: "Window too narrow for the preview",
		AppTabNew:           " new",
		AppModeHome:         "HOME",
		AppModeExplorer:     "EXPLORER",
		AppModePreview:      "PREVIEW",
		AppCmdlineTitle:     "Cmdline",
		AppSearchTitle:      "Search",

		AppWords + one:   "󰈭 %d word  ",
		AppWords + other: "󰈭 %d words  ",

		AppLnCol:             " Ln %d, Col %d ",
		AppNoFile:            "No file",
		AppUnsavedMark:       "unsaved ",
		AppTableHint:         "Tab next cell · / add row or column",
		AppErrorHint:         "  Esc dismiss · :messages",
		AppUntitled:          "Untitled-%d",
		AppUnsavedTitle:      "Unsaved changes",
		AppUnsavedQuestion:   "Save changes to %s before closing?",
		AppUnsavedLost:       "Your changes will be lost if you don't save them.",
		AppDontSave:          "Don't Save",
		AppGoneTitle:         "Not on disk",
		AppGoneWhat:          "%s was deleted or renamed outside Misoku.",
		AppGoneAsk:           "Recreate it at its old path with your version?",
		AppRecreate:          "Recreate",
		AppNotSavedKeep:      "Not saved: the version on disk could not be kept (%s)",
		AppBothKept:          "Both kept: %s",
		AppSaveFailed:        "Save failed: %s",
		AppWritten:           "\"%s\" %dL written",
		AppGoneSaveAsks:      "%q was deleted or renamed outside Misoku; saving will ask before recreating it",
		AppQuitMore:          "%s and %d more",
		AppQuitTitle:         "Quit Misoku?",
		AppQuitUnsaved:       "%d unsaved: %s",
		AppSaveAll:           "Save All",
		AppChangedOnDiskKept: "%q changed on disk; saving keeps both versions",
		AppReloaded:          "%q reloaded: it changed on disk",
	})
	register(ES, map[Key]string{
		AppNotLeaderKey:     "%s: no es una tecla líder",
		AppNeedsOpenNote:    "%s: necesita una nota abierta",
		AppNoNoteForLink:    "No hay ninguna nota para este enlace",
		AppLearnNotSaved:    "No se pudo recordar la respuesta (%s); Misoku volverá a preguntar la próxima vez",
		AppFileDialogFailed: "Falló el diálogo de archivos: %s",
		AppNoFileName:       "E32: No hay nombre de archivo",
		AppRenameFailed:     "No se pudo renombrar: %s",
		AppRenamedTo:        "Renombrado a %s",
		AppDeleteFile:       "El archivo se mueve a la Papelera.",
		AppDeleteFolder:     "La carpeta y todo lo que contiene se mueven a la Papelera.",
		AppDeleteNote:       "La nota y todas sus subnotas se mueven a la Papelera.",
		AppDeleteFileGone:   "El archivo se borra del disco.",
		AppDeleteFolderGone: "La carpeta y todo lo que contiene se borran del disco.",
		AppDeleteNoteGone:   "La nota y todas sus subnotas se borran del disco.",
		AppDeleteRestore:    "Puedes restaurarlo desde allí.",
		AppDeleteForGood:    "No se puede deshacer.",
		AppDeleteTitle:      "¿Eliminar %s?",
		AppDeleteFailed:     "No se pudo eliminar: %s",
		AppDeleted:          "Eliminado %s",
		AppLoading:          "cargando…",
		AppTooSmall:         "Terminal demasiado pequeña (%dx%d)\nAgrándala al menos a %dx%d",
		AppTooNarrowPreview: "La ventana es demasiado estrecha para la vista previa",
		AppTabNew:           " nueva",
		AppModeHome:         "INICIO",
		AppModeExplorer:     "EXPLORADOR",
		AppModePreview:      "VISTA PREVIA",
		AppCmdlineTitle:     "Comando",
		AppSearchTitle:      "Buscar",

		AppWords + one:   "󰈭 %d palabra  ",
		AppWords + other: "󰈭 %d palabras  ",

		AppLnCol:             " Lín %d, Col %d ",
		AppNoFile:            "Sin archivo",
		AppUnsavedMark:       "sin guardar ",
		AppTableHint:         "Tab celda siguiente · / añade fila o columna",
		AppErrorHint:         "  Esc descarta · :messages",
		AppUntitled:          "Sin título-%d",
		AppUnsavedTitle:      "Cambios sin guardar",
		AppUnsavedQuestion:   "¿Guardar los cambios de %s antes de cerrar?",
		AppUnsavedLost:       "Si no los guardas, perderás tus cambios.",
		AppDontSave:          "No guardar",
		AppGoneTitle:         "No está en el disco",
		AppGoneWhat:          "%s se borró o se renombró fuera de Misoku.",
		AppGoneAsk:           "¿Volver a crearlo en su ruta anterior con tu versión?",
		AppRecreate:          "Volver a crear",
		AppNotSavedKeep:      "No se guardó: no se pudo conservar la versión del disco (%s)",
		AppBothKept:          "Se conservaron las dos: %s",
		AppSaveFailed:        "No se pudo guardar: %s",
		AppWritten:           "\"%s\" %dL escritas",
		AppGoneSaveAsks:      "%q se borró o se renombró fuera de Misoku; al guardar se preguntará antes de volver a crearlo",
		AppQuitMore:          "%s y %d más",
		AppQuitTitle:         "¿Salir de Misoku?",
		AppQuitUnsaved:       "%d sin guardar: %s",
		AppSaveAll:           "Guardar todo",
		AppChangedOnDiskKept: "%q cambió en el disco; al guardar se conservan las dos versiones",
		AppReloaded:          "%q se recargó: cambió en el disco",
	})
}

// Dates in the date picker: weekdays from Sunday, months from January.
var (
	AppWeekdays    = [7]Key{"app.day.sun", "app.day.mon", "app.day.tue", "app.day.wed", "app.day.thu", "app.day.fri", "app.day.sat"}
	AppWeekdaysTwo = [7]Key{"app.day2.sun", "app.day2.mon", "app.day2.tue", "app.day2.wed", "app.day2.thu", "app.day2.fri", "app.day2.sat"}
	AppMonths      = [12]Key{"app.month.1", "app.month.2", "app.month.3", "app.month.4", "app.month.5", "app.month.6", "app.month.7", "app.month.8", "app.month.9", "app.month.10", "app.month.11", "app.month.12"}
	AppMonthsShort = [12]Key{"app.mon.1", "app.mon.2", "app.mon.3", "app.mon.4", "app.mon.5", "app.mon.6", "app.mon.7", "app.mon.8", "app.mon.9", "app.mon.10", "app.mon.11", "app.mon.12"}
)

func init() {
	for _, set := range []struct {
		keys   []Key
		en, es []string
	}{
		{AppWeekdays[:], []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}, []string{"dom", "lun", "mar", "mié", "jue", "vie", "sáb"}},
		{AppWeekdaysTwo[:], []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}, []string{"Do", "Lu", "Ma", "Mi", "Ju", "Vi", "Sá"}},
		{AppMonths[:],
			[]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
			[]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}},
		{AppMonthsShort[:],
			[]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
			[]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"}},
	} {
		en, es := map[Key]string{}, map[Key]string{}
		for i, k := range set.keys {
			en[k], es[k] = set.en[i], set.es[i]
		}
		register(EN, en)
		register(ES, es)
	}
}

// Notes, links, files, the finder and other screens.
const (
	AppDateLong          Key = "app.date.long"
	AppDateMonth         Key = "app.date.month"
	AppCopiedChars       Key = "app.copied_chars"
	AppCopiedLines       Key = "app.copied_lines"
	AppDailyDraft        Key = "app.daily_draft"
	AppSaveAsRootHint    Key = "app.save_as.root_hint"
	AppSaveAsNameHint    Key = "app.save_as.name_hint"
	AppSaveCancelled     Key = "app.save_cancelled"
	AppSaveAsTitle       Key = "app.save_as.title"
	AppSaveAsFolder      Key = "app.save_as.folder"
	AppSaveAsNewFolder   Key = "app.save_as.new_folder"
	AppSaveAsRoot        Key = "app.save_as.root"
	AppSaveAsName        Key = "app.save_as.name"
	AppSaveAsHelp        Key = "app.save_as.help"
	AppExportNoNote      Key = "app.export.no_note"
	AppExportFailed      Key = "app.export.failed"
	AppExportedStatus    Key = "app.export.status"
	AppExportedToast     Key = "app.export.toast"
	AppMovedNotes        Key = "app.moved_notes"
	AppNeedsNvim         Key = "app.needs_nvim"
	AppMiniFilesFailed   Key = "app.mini_files.failed"
	AppMiniFilesMissing  Key = "app.mini_files.missing"
	AppNeovimError       Key = "app.neovim_error"
	AppLinkFinderHelp    Key = "app.finder.link_help"
	AppFinderNoNotes     Key = "app.finder.no_notes"
	AppFinderNoLinks     Key = "app.finder.no_links"
	AppFinderHelp        Key = "app.finder.help"
	AppOpenedNewTab      Key = "app.finder.opened_new_tab"
	AppFinderLinkTitle   Key = "app.finder.link_title"
	AppFinderTitle       Key = "app.finder.title"
	AppFinderPlaceholder Key = "app.finder.placeholder"
	AppGraphTab          Key = "app.graph_tab"
	AppGlobalGraphTab    Key = "app.global_graph_tab"
	AppHistoryOf         Key = "app.history.of"
	AppHistoryHelp       Key = "app.history.help"
	AppHistoryNoNote     Key = "app.history.no_note"
	AppHistoryNeverSaved Key = "app.history.never_saved"
	AppHistoryReading    Key = "app.history.reading"
	AppHistoryBadVersion Key = "app.history.bad_version"
	AppHistoryNoGit      Key = "app.history.no_git"
	AppHistoryNoRepo     Key = "app.history.no_repo"
	AppHistoryUnreadable Key = "app.history.unreadable"
	AppHistoryUntracked  Key = "app.history.untracked"
	AppHistoryStill      Key = "app.history.still"
	AppHistorySame       Key = "app.history.same"
	AppHistoryRestored   Key = "app.history.restored"
	AppReading           Key = "app.reading"
	AppNoLinksInNote     Key = "app.no_links_in_note"
	AppLeaderFooter      Key = "app.leader.footer"
	AppLeaderEnlarge     Key = "app.leader.enlarge"
	AppLeaderThen        Key = "app.leader.then"
	AppNoSection         Key = "app.no_section"
	AppNewNoteDraft      Key = "app.new_note_draft"
	AppOpenTitle         Key = "app.open.title"
	AppOpenWhy           Key = "app.open.why"
	AppCouldNotOpen      Key = "app.could_not_open"
	AppOpened            Key = "app.opened"
	AppNoLinkUnder       Key = "app.no_link_under"
	AppNoPreviousNote    Key = "app.no_previous_note"
	AppUnknownBorder     Key = "app.unknown_border"
	AppChooseImage       Key = "app.choose.image"
	AppChooseVideo       Key = "app.choose.video"
	AppChooseAudio       Key = "app.choose.audio"
	AppChooseFile        Key = "app.choose.file"
	AppChooseCover       Key = "app.choose.cover"
	AppKindImage         Key = "app.kind.image"
	AppKindVideo         Key = "app.kind.video"
	AppKindAudio         Key = "app.kind.audio"
	AppKindFile          Key = "app.kind.file"
	AppKindCover         Key = "app.kind.cover"
	AppDialogOpened      Key = "app.dialog_opened"
	AppAllFiles          Key = "app.all_files"
	AppAttached          Key = "app.attached"
	AppPasted            Key = "app.pasted"
	AppOpenNoteFirst     Key = "app.open_note_first"
	AppAttachFailed      Key = "app.attach_failed"
	AppCoverSet          Key = "app.cover_set"
	AppSavedWithFolder   Key = "app.saved_with_folder"
	AppBrowserFilter     Key = "app.browser.filter"
	AppBrowserNotA       Key = "app.browser.not_a"
	AppBrowserEmpty      Key = "app.browser.empty"
	AppBrowserHelp       Key = "app.browser.help"
	AppMenuFilter        Key = "app.menu.filter"
	AppMenuNoItem        Key = "app.menu.no_item"
	AppMenuNumber        Key = "app.menu.number"
	AppMenuSearchHint    Key = "app.menu.search_hint"
	AppMessagesNone      Key = "app.messages.none"
	AppMessagesTitle     Key = "app.messages.title"
	AppCantCreate        Key = "app.cant_create"
	AppExistsOpened      Key = "app.exists_opened"
	AppExists            Key = "app.exists"
	AppCreatesNote       Key = "app.creates_note"
	AppCreatesFolder     Key = "app.creates_folder"
	AppCreateTitle       Key = "app.create_title"
	AppCreateFailed      Key = "app.create_failed"
	AppCreated           Key = "app.created"
)

func init() {
	register(EN, map[Key]string{
		AppDateLong:          "%s, %d %s %d",
		AppDateMonth:         "%s %d",
		AppCopiedChars:       "Copied %d characters",
		AppCopiedLines:       "Copied %d lines",
		AppDailyDraft:        "New daily note \"%s\": save to create it in %s",
		AppSaveAsRootHint:    "/  (workspace root)",
		AppSaveAsNameHint:    "note name",
		AppSaveCancelled:     "Save cancelled",
		AppSaveAsTitle:       "Save As",
		AppSaveAsFolder:      "Folder",
		AppSaveAsNewFolder:   "     new folder will be created",
		AppSaveAsRoot:        "/  workspace root",
		AppSaveAsName:        "Name",
		AppSaveAsHelp:        "Tab switch field · ↑↓ folder · Enter save · Esc cancel",
		AppExportNoNote:      "Open a note to export it",
		AppExportFailed:      "Export failed: %s",
		AppExportedStatus:    "\"%s\" exported",
		AppExportedToast:     "Exported %s",
		AppMovedNotes:        "Moved %d notes",
		AppNeedsNvim:         "%s needs Neovim (nvim) on PATH",
		AppMiniFilesFailed:   "Could not start mini.files: %s",
		AppMiniFilesMissing:  "mini.files is not installed in Neovim (add nvim-mini/mini.files, or the LazyVim extra editor.mini-files)",
		AppNeovimError:       "Neovim: %s",
		AppLinkFinderHelp:    "# heading · ^ block · Esc type it yourself",
		AppFinderNoNotes:     "   No matching notes",
		AppFinderNoLinks:     "   No matching links",
		AppFinderHelp:        "↑↓ move · Enter open here (jumps to the line) · Ctrl+T new tab · Esc close",
		AppOpenedNewTab:      "Opened in a new tab — %s has unsaved changes",
		AppFinderLinkTitle:   "Link to Note",
		AppFinderTitle:       "Find Note",
		AppFinderPlaceholder: "Search file names and text in every note…",
		AppGraphTab:          "Graph",
		AppGlobalGraphTab:    "Global graph",
		AppHistoryOf:         "History of %s",
		AppHistoryHelp:       " j/k move · Enter restore into the editor (u undoes it) · Esc close",
		AppHistoryNoNote:     "History needs an open note",
		AppHistoryNeverSaved: "This note was never saved, so it has no history yet",
		AppHistoryReading:    "Reading the history of %s…",
		AppHistoryBadVersion: "Could not read this version: %s",
		AppHistoryNoGit:      "History needs git, and git is not installed",
		AppHistoryNoRepo:     "No history: this vault is not a git repository",
		AppHistoryUnreadable: "Could not read the history: %s",
		AppHistoryUntracked:  "%s is not in git yet: it was never committed",
		AppHistoryStill:      "Still reading this version…",
		AppHistorySame:       "The note already holds the version of %s",
		AppHistoryRestored:   "Restored the version of %s (%s): u undoes it, save to keep it",
		AppReading:           "Reading…",
		AppNoLinksInNote:     "No links in this note",
		AppLeaderFooter:      " These keys come from config.json · Esc to close",
		AppLeaderEnlarge:     " … enlarge the terminal to see all",
		AppLeaderThen:        " then",
		AppNoSection:         "No section \"%s\" in this note",
		AppNewNoteDraft:      "New note \"%s\": save to create it",
		AppOpenTitle:         "Open %s?",
		AppOpenWhy:           "The system may run it, or it is outside the vault.",
		AppCouldNotOpen:      "Could not open %s: %s",
		AppOpened:            "Opened %s",
		AppNoLinkUnder:       "No link under the cursor",
		AppNoPreviousNote:    "No previous note in this tab",
		AppUnknownBorder:     "ui: unknown border %q (use rounded or single)",
		AppChooseImage:       "Choose an image",
		AppChooseVideo:       "Choose a video",
		AppChooseAudio:       "Choose an audio file",
		AppChooseFile:        "Choose a file",
		AppChooseCover:       "Choose a cover image",
		AppKindImage:         "image",
		AppKindVideo:         "video",
		AppKindAudio:         "audio",
		AppKindFile:          "file",
		AppKindCover:         "cover",
		AppDialogOpened:      "%s… (a file dialog opened)",
		AppAllFiles:          "All files",
		AppAttached:          "Attached %s",
		AppPasted:            "Pasted %s",
		AppOpenNoteFirst:     "Open a note first",
		AppAttachFailed:      "Could not attach file: %s",
		AppCoverSet:          "Cover set to %s",
		AppSavedWithFolder:   "%s — saved with the note's folder once you save",
		AppBrowserFilter:     "Filter, or type a path and press Enter",
		AppBrowserNotA:       "Not a %s: %s",
		AppBrowserEmpty:      "   Nothing here",
		AppBrowserHelp:       " Enter open/choose · ← or Backspace up · ~/path jumps · Esc cancel",
		AppMenuFilter:        "Type to filter (/ for digits)…",
		AppMenuNoItem:        " No item %s · Backspace or Esc",
		AppMenuNumber:        " Number: %s_ · Enter picks it",
		AppMenuSearchHint:    " Type or / to filter · ↑↓ Enter · Esc",
		AppMessagesNone:      "No messages yet",
		AppMessagesTitle:     "Messages",
		AppCantCreate:        "Can't create %s: %s",
		AppExistsOpened:      "%s already exists: opened it",
		AppExists:            "%s already exists",
		AppCreatesNote:       "Creates the note and opens it.",
		AppCreatesFolder:     "Creates the folder.",
		AppCreateTitle:       "CREATE %s",
		AppCreateFailed:      "Create failed: %s",
		AppCreated:           "Created %s",
	})
	register(ES, map[Key]string{
		AppDateLong:          "%s, %d %s %d",
		AppDateMonth:         "%s %d",
		AppCopiedChars:       "Se copiaron %d caracteres",
		AppCopiedLines:       "Se copiaron %d líneas",
		AppDailyDraft:        "Nueva nota diaria \"%s\": guárdala para crearla en %s",
		AppSaveAsRootHint:    "/  (raíz del espacio de trabajo)",
		AppSaveAsNameHint:    "nombre de la nota",
		AppSaveCancelled:     "Guardado cancelado",
		AppSaveAsTitle:       "Guardar como",
		AppSaveAsFolder:      "Carpeta",
		AppSaveAsNewFolder:   "     se creará una carpeta nueva",
		AppSaveAsRoot:        "/  raíz del espacio de trabajo",
		AppSaveAsName:        "Nombre",
		AppSaveAsHelp:        "Tab cambia de campo · ↑↓ carpeta · Enter guarda · Esc cancela",
		AppExportNoNote:      "Abre una nota para exportarla",
		AppExportFailed:      "No se pudo exportar: %s",
		AppExportedStatus:    "\"%s\" exportado",
		AppExportedToast:     "Exportado %s",
		AppMovedNotes:        "Se movieron %d notas",
		AppNeedsNvim:         "%s necesita Neovim (nvim) en el PATH",
		AppMiniFilesFailed:   "No se pudo iniciar mini.files: %s",
		AppMiniFilesMissing:  "mini.files no está instalado en Neovim (añade nvim-mini/mini.files, o el extra de LazyVim editor.mini-files)",
		AppNeovimError:       "Neovim: %s",
		AppLinkFinderHelp:    "# título · ^ bloque · Esc lo escribes tú",
		AppFinderNoNotes:     "   Ninguna nota coincide",
		AppFinderNoLinks:     "   Ningún enlace coincide",
		AppFinderHelp:        "↑↓ mueve · Enter abre aquí (salta a la línea) · Ctrl+T pestaña nueva · Esc cierra",
		AppOpenedNewTab:      "Se abrió en una pestaña nueva — %s tiene cambios sin guardar",
		AppFinderLinkTitle:   "Enlazar a una nota",
		AppFinderTitle:       "Buscar nota",
		AppFinderPlaceholder: "Busca nombres de archivo y texto en todas las notas…",
		AppGraphTab:          "Grafo",
		AppGlobalGraphTab:    "Grafo global",
		AppHistoryOf:         "Historial de %s",
		AppHistoryHelp:       " j/k mueve · Enter restaura en el editor (u lo deshace) · Esc cierra",
		AppHistoryNoNote:     "El historial necesita una nota abierta",
		AppHistoryNeverSaved: "Esta nota nunca se guardó, así que aún no tiene historial",
		AppHistoryReading:    "Leyendo el historial de %s…",
		AppHistoryBadVersion: "No se pudo leer esta versión: %s",
		AppHistoryNoGit:      "El historial necesita git, y git no está instalado",
		AppHistoryNoRepo:     "Sin historial: este vault no es un repositorio git",
		AppHistoryUnreadable: "No se pudo leer el historial: %s",
		AppHistoryUntracked:  "%s aún no está en git: nunca se hizo commit",
		AppHistoryStill:      "Todavía se está leyendo esta versión…",
		AppHistorySame:       "La nota ya tiene la versión del %s",
		AppHistoryRestored:   "Se restauró la versión del %s (%s): u lo deshace, guarda para conservarla",
		AppReading:           "Leyendo…",
		AppNoLinksInNote:     "No hay enlaces en esta nota",
		AppLeaderFooter:      " Estas teclas vienen de config.json · Esc cierra",
		AppLeaderEnlarge:     " … agranda la terminal para verlas todas",
		AppLeaderThen:        " y luego",
		AppNoSection:         "No hay ninguna sección \"%s\" en esta nota",
		AppNewNoteDraft:      "Nota nueva \"%s\": guárdala para crearla",
		AppOpenTitle:         "¿Abrir %s?",
		AppOpenWhy:           "El sistema podría ejecutarlo, o está fuera del vault.",
		AppCouldNotOpen:      "No se pudo abrir %s: %s",
		AppOpened:            "Abierto %s",
		AppNoLinkUnder:       "No hay ningún enlace bajo el cursor",
		AppNoPreviousNote:    "No hay una nota anterior en esta pestaña",
		AppUnknownBorder:     "ui: borde desconocido %q (usa rounded o single)",
		AppChooseImage:       "Elige una imagen",
		AppChooseVideo:       "Elige un video",
		AppChooseAudio:       "Elige un archivo de audio",
		AppChooseFile:        "Elige un archivo",
		AppChooseCover:       "Elige una imagen de portada",
		AppKindImage:         "imagen",
		AppKindVideo:         "video",
		AppKindAudio:         "audio",
		AppKindFile:          "archivo",
		AppKindCover:         "portada",
		AppDialogOpened:      "%s… (se abrió un diálogo de archivos)",
		AppAllFiles:          "Todos los archivos",
		AppAttached:          "Adjuntado %s",
		AppPasted:            "Pegado %s",
		AppOpenNoteFirst:     "Primero abre una nota",
		AppAttachFailed:      "No se pudo adjuntar el archivo: %s",
		AppCoverSet:          "Portada: %s",
		AppSavedWithFolder:   "%s — se guarda en la carpeta de la nota cuando la guardes",
		AppBrowserFilter:     "Filtra, o escribe una ruta y pulsa Enter",
		AppBrowserNotA:       "No es un archivo de tipo %s: %s",
		AppBrowserEmpty:      "   Aquí no hay nada",
		AppBrowserHelp:       " Enter abre/elige · ← o Retroceso sube · ~/ruta salta · Esc cancela",
		AppMenuFilter:        "Escribe para filtrar (/ para dígitos)…",
		AppMenuNoItem:        " No hay elemento %s · Retroceso o Esc",
		AppMenuNumber:        " Número: %s_ · Enter lo elige",
		AppMenuSearchHint:    " Escribe o / para filtrar · ↑↓ Enter · Esc",
		AppMessagesNone:      "Todavía no hay mensajes",
		AppMessagesTitle:     "Mensajes",
		AppCantCreate:        "No se puede crear %s: %s",
		AppExistsOpened:      "%s ya existe: se abrió",
		AppExists:            "%s ya existe",
		AppCreatesNote:       "Crea la nota y la abre.",
		AppCreatesFolder:     "Crea la carpeta.",
		AppCreateTitle:       "CREAR %s",
		AppCreateFailed:      "No se pudo crear: %s",
		AppCreated:           "Creado %s",
	})
}

// The command palette's fixed keys: the editor, the preview, the graphs
// and the mouse, each with its line and its explanation.
const (
	AppPal01       Key = "app.palette.01"
	AppPal01Long   Key = "app.palette.01.long"
	AppPal02       Key = "app.palette.02"
	AppPal02Long   Key = "app.palette.02.long"
	AppPal03       Key = "app.palette.03"
	AppPal03Long   Key = "app.palette.03.long"
	AppPal04       Key = "app.palette.04"
	AppPal04Long   Key = "app.palette.04.long"
	AppPal05       Key = "app.palette.05"
	AppPal05Long   Key = "app.palette.05.long"
	AppPal06       Key = "app.palette.06"
	AppPal06Long   Key = "app.palette.06.long"
	AppPal07       Key = "app.palette.07"
	AppPal07Long   Key = "app.palette.07.long"
	AppPal08       Key = "app.palette.08"
	AppPal08Long   Key = "app.palette.08.long"
	AppPal09       Key = "app.palette.09"
	AppPal09Long   Key = "app.palette.09.long"
	AppPal10       Key = "app.palette.10"
	AppPal10Long   Key = "app.palette.10.long"
	AppPal11       Key = "app.palette.11"
	AppPal11Long   Key = "app.palette.11.long"
	AppPal12       Key = "app.palette.12"
	AppPal12Long   Key = "app.palette.12.long"
	AppPal13       Key = "app.palette.13"
	AppPal13Long   Key = "app.palette.13.long"
	AppPal14       Key = "app.palette.14"
	AppPal14Long   Key = "app.palette.14.long"
	AppPal15       Key = "app.palette.15"
	AppPal15Long   Key = "app.palette.15.long"
	AppPal16       Key = "app.palette.16"
	AppPal16Long   Key = "app.palette.16.long"
	AppPal17       Key = "app.palette.17"
	AppPal17Long   Key = "app.palette.17.long"
	AppPal18       Key = "app.palette.18"
	AppPal18Long   Key = "app.palette.18.long"
	AppPal19       Key = "app.palette.19"
	AppPal19Long   Key = "app.palette.19.long"
	AppPal20       Key = "app.palette.20"
	AppPal20Long   Key = "app.palette.20.long"
	AppPal21       Key = "app.palette.21"
	AppPal21Long   Key = "app.palette.21.long"
	AppPal22       Key = "app.palette.22"
	AppPal22Long   Key = "app.palette.22.long"
	AppPal23       Key = "app.palette.23"
	AppPal23Long   Key = "app.palette.23.long"
	AppPal24       Key = "app.palette.24"
	AppPal24Long   Key = "app.palette.24.long"
	AppPal25       Key = "app.palette.25"
	AppPal25Long   Key = "app.palette.25.long"
	AppPal26       Key = "app.palette.26"
	AppPal26Long   Key = "app.palette.26.long"
	AppPal27       Key = "app.palette.27"
	AppPal27Long   Key = "app.palette.27.long"
	AppPal28       Key = "app.palette.28"
	AppPal28Long   Key = "app.palette.28.long"
	AppPal29       Key = "app.palette.29"
	AppPal29Long   Key = "app.palette.29.long"
	AppPal30       Key = "app.palette.30"
	AppPal30Long   Key = "app.palette.30.long"
	AppPal31       Key = "app.palette.31"
	AppPal31Long   Key = "app.palette.31.long"
	AppPal32       Key = "app.palette.32"
	AppPal32Long   Key = "app.palette.32.long"
	AppPal33       Key = "app.palette.33"
	AppPal33Long   Key = "app.palette.33.long"
	AppPal34       Key = "app.palette.34"
	AppPal34Long   Key = "app.palette.34.long"
	AppPal35       Key = "app.palette.35"
	AppPal35Long   Key = "app.palette.35.long"
	AppPal36       Key = "app.palette.36"
	AppPal36Long   Key = "app.palette.36.long"
	AppPal37       Key = "app.palette.37"
	AppPal37Long   Key = "app.palette.37.long"
	AppPal38       Key = "app.palette.38"
	AppPal38Long   Key = "app.palette.38.long"
	AppPal39       Key = "app.palette.39"
	AppPal39Long   Key = "app.palette.39.long"
	AppPal40       Key = "app.palette.40"
	AppPal40Long   Key = "app.palette.40.long"
	AppPal41       Key = "app.palette.41"
	AppPal41Long   Key = "app.palette.41.long"
	AppPal42       Key = "app.palette.42"
	AppPal42Long   Key = "app.palette.42.long"
	AppPal43       Key = "app.palette.43"
	AppPal43Long   Key = "app.palette.43.long"
	AppPal44       Key = "app.palette.44"
	AppPal44Long   Key = "app.palette.44.long"
	AppPalKeyClick Key = "app.palette.key.click"
	AppPalKeyWheel Key = "app.palette.key.wheel"
	AppPalKeyDrag  Key = "app.palette.key.drag"
)

func init() {
	register(EN, map[Key]string{
		AppPal01:       "Insert mode",
		AppPal01Long:   "Starts typing: before the cursor (i), after it (a), at the end of the line (A) or at its start (I). Esc goes back to NORMAL.",
		AppPal02:       "Indent / outdent (insert)",
		AppPal02Long:   "In INSERT mode, Tab indents the line or the list item and Shift+Tab takes the indent back; in a table they move between cells.",
		AppPal03:       "New line below / above",
		AppPal03Long:   "Opens a new line below (o) or above (O) the cursor and starts typing on it.",
		AppPal04:       "Move",
		AppPal04Long:   "Moves the cursor left, down, up and right, as the arrow keys do.",
		AppPal05:       "Next / previous word",
		AppPal05Long:   "Moves the cursor to the start of the next word (w) or of the previous one (b).",
		AppPal06:       "Line start / end",
		AppPal06Long:   "Moves the cursor to the start (0) or the end ($) of the line.",
		AppPal07:       "Top / bottom",
		AppPal07Long:   "Moves the cursor to the first (gg) or the last (G) line of the note.",
		AppPal08:       "Half page down / up",
		AppPal08Long:   "Scrolls the editor half a screen down (Ctrl+D) or up (Ctrl+U).",
		AppPal09:       "Delete char / line",
		AppPal09Long:   "Deletes the character under the cursor (x) or the whole line (dd), which can then be pasted with p.",
		AppPal10:       "Copy line / paste",
		AppPal10Long:   "Copies the line (yy) and pastes what was copied or deleted below the cursor (p).",
		AppPal11:       "Select, then y copy, d cut",
		AppPal11Long:   "Starts selecting by character (v) or by line (V); then y copies the selection and d cuts it.",
		AppPal12:       "Undo",
		AppPal12Long:   "Undoes the last change: a whole stay in INSERT mode, or one NORMAL command such as dd, p or x.",
		AppPal13:       "Redo",
		AppPal13Long:   "Redoes the last change that u undid.",
		AppPal14:       "Block menu: headings, lists, blocks",
		AppPal14Long:   "Typed in INSERT mode at the start of a line or after a space, / opens the block menu: headings, lists, tables, callouts, media and more.",
		AppPal15:       "Link to a note",
		AppPal15Long:   "Typed in INSERT mode, [[ opens the note finder to pick the note to link to.",
		AppPal16:       "Follow the link under the cursor",
		AppPal16Long:   "Opens the note or the web page the link under the cursor points to.",
		AppPal17:       "Follow [[link]] / toggle box",
		AppPal17Long:   "In NORMAL mode, follows the [[link]] under the cursor (not inside code; a note that does not exist is not started), toggles the box of a checkbox line, and otherwise moves down a line.",
		AppPal18:       "Command line",
		AppPal18Long:   "Opens the command line in a box near the top of the screen for the commands below; Enter runs it, Esc leaves it.",
		AppPal19:       "Search in the note",
		AppPal19Long:   "Opens the search prompt in a box near the top of the screen: Enter finds the next match of the text from the cursor, and n or N repeat it forward or backward, wrapping around the note. The text is literal, not a regex, and the search ignores case unless it has an upper-case letter; Esc leaves it.",
		AppPal20:       "Next / previous match",
		AppPal20Long:   "After a search, n goes to the next match and N to the previous one, both wrapping around the note and saying so in the status bar. Without a search they say there is none.",
		AppPal21:       "Scroll a line",
		AppPal21Long:   "With the focus on the preview, scrolls it one line down (j) or up (k).",
		AppPal22:       "Half page down / up",
		AppPal22Long:   "With the focus on the preview, scrolls it half a screen down (d) or up (u); Ctrl+D and Ctrl+U too.",
		AppPal23:       "Page down",
		AppPal23Long:   "With the focus on the preview, scrolls it a whole screen down; PgUp goes back up.",
		AppPal24:       "Top / bottom",
		AppPal24Long:   "With the focus on the preview, jumps to its top (g) or its bottom (G).",
		AppPal25:       "Calendar month back / forward",
		AppPal25Long:   "Moves the calendar blocks of the note one month back (<, or H) or forward (>, or L).",
		AppPal26:       "Calendar: this month",
		AppPal26Long:   "Brings the calendar blocks of the note back to the current month.",
		AppPal27:       "Fold / unfold all",
		AppPal27Long:   "Folds (zM) or unfolds (zR) every foldable block of the preview at once.",
		AppPal28:       "Move from note to note",
		AppPal28Long:   "In the graph, j and k move between notes at the same level; h and l move one column left or right, to the parent or to a child.",
		AppPal29:       "Open the selected note",
		AppPal29Long:   "In the graph, opens the selected note in the editor.",
		AppPal30:       "Zoom in / out",
		AppPal30Long:   "In the graph, shows the map closer (+) or further away (-).",
		AppPal31:       "Centre the graph",
		AppPal31Long:   "In the graph, puts the selected note back in the middle of the map.",
		AppPal32:       "Back",
		AppPal32Long:   "In the graph, goes back to the note shown before, as the back button does.",
		AppPal33:       "Move to the nearest note",
		AppPal33Long:   "In the global graph, selects the nearest note to the left, below, above or to the right; the arrow keys do the same.",
		AppPal34:       "Next / previous link",
		AppPal34Long:   "In the global graph, selects the next (n) or the previous (N) note the selected one links to or is linked from.",
		AppPal35:       "Open the note, or start a missing one",
		AppPal35Long:   "In the global graph, opens the selected note in the editor; on a link to a note that doesn't exist yet, starts that note, unsaved until you save it.",
		AppPal36:       "Zoom: dots, busy names, all names",
		AppPal36Long:   "In the global graph, + shows the map closer, with more names, and - further away; the wheel zooms around the mouse.",
		AppPal37:       "Centre on the selection",
		AppPal37Long:   "In the global graph, moves the view so the selected note is in the middle.",
		AppPal38:       "Back to the previous selection",
		AppPal38Long:   "In the global graph, clears the filter first; without one, selects the note selected before, as the back button does.",
		AppPal39:       "Filter the map by name or path",
		AppPal39Long:   "In the global graph, greys out the notes whose name or path doesn't hold the text typed; Enter selects the best match and keeps the filter, Esc clears it.",
		AppPal40:       "Hide / show notes without links",
		AppPal40Long:   "In the global graph, hides the notes that link nowhere and that nothing links to, or shows them again. They keep their place on the map.",
		AppPal41:       "Hide / show missing notes",
		AppPal41Long:   "In the global graph, hides the links to notes that don't exist yet, or shows them again.",
		AppPal42:       "Tabs, tree, cursor",
		AppPal42Long:   "A click picks a tab, a row of the explorer, a place in the editor, or a checkbox or a link in the preview.",
		AppPal43:       "Scroll any pane",
		AppPal43Long:   "The wheel scrolls the pane under the mouse, whichever has the focus.",
		AppPal44:       "Resize the panes",
		AppPal44Long:   "Dragging a divider between two panes makes one wider and the other narrower.",
		AppPalKeyClick: "Click",
		AppPalKeyWheel: "Wheel",
		AppPalKeyDrag:  "Drag",
	})
	register(ES, map[Key]string{
		AppPal01:       "Modo INSERT",
		AppPal01Long:   "Empieza a escribir: antes del cursor (i), después (a), al final de la línea (A) o al principio (I). Esc vuelve a NORMAL.",
		AppPal02:       "Sangrar / quitar sangría (insert)",
		AppPal02Long:   "En modo INSERT, Tab sangra la línea o el elemento de la lista y Shift+Tab quita la sangría; en una tabla se mueven entre celdas.",
		AppPal03:       "Línea nueva debajo / encima",
		AppPal03Long:   "Abre una línea nueva debajo (o) o encima (O) del cursor y empieza a escribir en ella.",
		AppPal04:       "Mover",
		AppPal04Long:   "Mueve el cursor a la izquierda, abajo, arriba y a la derecha, como las flechas.",
		AppPal05:       "Palabra siguiente / anterior",
		AppPal05Long:   "Lleva el cursor al principio de la palabra siguiente (w) o de la anterior (b).",
		AppPal06:       "Principio / final de línea",
		AppPal06Long:   "Lleva el cursor al principio (0) o al final ($) de la línea.",
		AppPal07:       "Arriba del todo / abajo del todo",
		AppPal07Long:   "Lleva el cursor a la primera (gg) o a la última (G) línea de la nota.",
		AppPal08:       "Media página abajo / arriba",
		AppPal08Long:   "Desplaza el editor media pantalla hacia abajo (Ctrl+D) o hacia arriba (Ctrl+U).",
		AppPal09:       "Borrar carácter / línea",
		AppPal09Long:   "Borra el carácter bajo el cursor (x) o la línea entera (dd), que luego se puede pegar con p.",
		AppPal10:       "Copiar línea / pegar",
		AppPal10Long:   "Copia la línea (yy) y pega lo copiado o borrado debajo del cursor (p).",
		AppPal11:       "Seleccionar, luego y copia, d corta",
		AppPal11Long:   "Empieza a seleccionar por carácter (v) o por línea (V); luego y copia la selección y d la corta.",
		AppPal12:       "Deshacer",
		AppPal12Long:   "Deshace el último cambio: todo un paso por el modo INSERT, o un comando de NORMAL como dd, p o x.",
		AppPal13:       "Rehacer",
		AppPal13Long:   "Rehace el último cambio que u deshizo.",
		AppPal14:       "Menú de bloques: títulos, listas, bloques",
		AppPal14Long:   "Escrito en modo INSERT al principio de una línea o tras un espacio, / abre el menú de bloques: títulos, listas, tablas, avisos, multimedia y más.",
		AppPal15:       "Enlazar a una nota",
		AppPal15Long:   "Escrito en modo INSERT, [[ abre el buscador de notas para elegir la nota a enlazar.",
		AppPal16:       "Seguir el enlace bajo el cursor",
		AppPal16Long:   "Abre la nota o la página web a la que apunta el enlace bajo el cursor.",
		AppPal17:       "Seguir [[enlace]] / marcar casilla",
		AppPal17Long:   "En modo NORMAL, sigue el [[enlace]] bajo el cursor (no dentro de código; una nota que no existe no se empieza), marca o desmarca la casilla de una línea con casilla y, si no, baja una línea.",
		AppPal18:       "Línea de comandos",
		AppPal18Long:   "Abre la línea de comandos en un recuadro cerca de la parte de arriba de la pantalla para los comandos de abajo; Enter la ejecuta, Esc sale.",
		AppPal19:       "Buscar en la nota",
		AppPal19Long:   "Abre la búsqueda en un recuadro cerca de la parte de arriba de la pantalla: Enter encuentra la siguiente coincidencia del texto desde el cursor, y n o N la repiten hacia delante o hacia atrás, dando la vuelta a la nota. El texto es literal, no una regex, y la búsqueda no distingue mayúsculas salvo que el texto tenga alguna; Esc sale.",
		AppPal20:       "Coincidencia siguiente / anterior",
		AppPal20Long:   "Tras una búsqueda, n va a la siguiente coincidencia y N a la anterior, ambas dando la vuelta a la nota y avisándolo en la barra de estado. Sin búsqueda, dicen que no hay ninguna.",
		AppPal21:       "Desplazar una línea",
		AppPal21Long:   "Con el foco en la vista previa, la desplaza una línea hacia abajo (j) o hacia arriba (k).",
		AppPal22:       "Media página abajo / arriba",
		AppPal22Long:   "Con el foco en la vista previa, la desplaza media pantalla hacia abajo (d) o hacia arriba (u); también Ctrl+D y Ctrl+U.",
		AppPal23:       "Página abajo",
		AppPal23Long:   "Con el foco en la vista previa, la desplaza una pantalla entera hacia abajo; PgUp vuelve arriba.",
		AppPal24:       "Arriba del todo / abajo del todo",
		AppPal24Long:   "Con el foco en la vista previa, salta a su principio (g) o a su final (G).",
		AppPal25:       "Mes del calendario anterior / siguiente",
		AppPal25Long:   "Mueve los bloques de calendario de la nota un mes atrás (<, o H) o adelante (>, o L).",
		AppPal26:       "Calendario: este mes",
		AppPal26Long:   "Devuelve los bloques de calendario de la nota al mes actual.",
		AppPal27:       "Plegar / desplegar todo",
		AppPal27Long:   "Pliega (zM) o despliega (zR) a la vez todos los bloques plegables de la vista previa.",
		AppPal28:       "Moverse de nota en nota",
		AppPal28Long:   "En el grafo, j y k se mueven entre notas del mismo nivel; h y l se mueven una columna a la izquierda o a la derecha, al padre o a un hijo.",
		AppPal29:       "Abrir la nota seleccionada",
		AppPal29Long:   "En el grafo, abre la nota seleccionada en el editor.",
		AppPal30:       "Acercar / alejar",
		AppPal30Long:   "En el grafo, muestra el mapa más cerca (+) o más lejos (-).",
		AppPal31:       "Centrar el grafo",
		AppPal31Long:   "En el grafo, vuelve a poner la nota seleccionada en el centro del mapa.",
		AppPal32:       "Atrás",
		AppPal32Long:   "En el grafo, vuelve a la nota que se mostraba antes, como el botón de volver.",
		AppPal33:       "Ir a la nota más cercana",
		AppPal33Long:   "En el grafo global, selecciona la nota más cercana a la izquierda, abajo, arriba o a la derecha; las flechas hacen lo mismo.",
		AppPal34:       "Enlace siguiente / anterior",
		AppPal34Long:   "En el grafo global, selecciona la nota siguiente (n) o la anterior (N) de las que enlaza la seleccionada o la enlazan.",
		AppPal35:       "Abrir la nota, o empezar una que falta",
		AppPal35Long:   "En el grafo global, abre la nota seleccionada en el editor; en un enlace a una nota que aún no existe, empieza esa nota, sin guardar hasta que la guardes.",
		AppPal36:       "Zoom: puntos, nombres concurridos, todos los nombres",
		AppPal36Long:   "En el grafo global, + muestra el mapa más cerca, con más nombres, y - más lejos; la rueda acerca alrededor del ratón.",
		AppPal37:       "Centrar en la selección",
		AppPal37Long:   "En el grafo global, mueve la vista para que la nota seleccionada quede en el centro.",
		AppPal38:       "Volver a la selección anterior",
		AppPal38Long:   "En el grafo global, primero quita el filtro; sin filtro, selecciona la nota seleccionada antes, como el botón de volver.",
		AppPal39:       "Filtrar el mapa por nombre o ruta",
		AppPal39Long:   "En el grafo global, atenúa las notas cuyo nombre o ruta no contiene el texto escrito; Enter selecciona la mejor coincidencia y mantiene el filtro, Esc lo quita.",
		AppPal40:       "Ocultar / mostrar notas sin enlaces",
		AppPal40Long:   "En el grafo global, oculta las notas que no enlazan a ninguna parte y a las que nada enlaza, o las vuelve a mostrar. Mantienen su sitio en el mapa.",
		AppPal41:       "Ocultar / mostrar notas que faltan",
		AppPal41Long:   "En el grafo global, oculta los enlaces a notas que aún no existen, o los vuelve a mostrar.",
		AppPal42:       "Pestañas, árbol, cursor",
		AppPal42Long:   "Un clic elige una pestaña, una fila del explorador, un sitio en el editor, o una casilla o un enlace en la vista previa.",
		AppPal43:       "Desplazar cualquier panel",
		AppPal43Long:   "La rueda desplaza el panel bajo el ratón, tenga o no el foco.",
		AppPal44:       "Cambiar el tamaño de los paneles",
		AppPal44Long:   "Arrastrar el divisor entre dos paneles ensancha uno y estrecha el otro.",
		AppPalKeyClick: "Clic",
		AppPalKeyWheel: "Rueda",
		AppPalKeyDrag:  "Arrastrar",
	})
}

// The command palette's groups and its own lines.
const (
	AppPalGroupExport   Key = "app.palette.group.export"
	AppPalGroupGeneral  Key = "app.palette.group.general"
	AppPalGroupExplorer Key = "app.palette.group.explorer"
	AppPalGroupEditor   Key = "app.palette.group.editor"
	AppPalGroupCommands Key = "app.palette.group.commands"
	AppPalGroupPreview  Key = "app.palette.group.preview"
	AppPalGroupGraph    Key = "app.palette.group.graph"
	AppPalGroupGlobal   Key = "app.palette.group.global"
	AppPalGroupMouse    Key = "app.palette.group.mouse"
	AppPalShell         Key = "app.palette.shell"
	AppPalShellLong     Key = "app.palette.shell.long"
	AppPalPlaceholder   Key = "app.palette.placeholder"
	AppPalNoMatch       Key = "app.palette.no_match"
	AppPalTitle         Key = "app.palette.title"
	AppPalHint          Key = "app.palette.hint"
	AppPalHintNeedsNote Key = "app.palette.hint_needs_note"
	AppPalHintKeyOnly   Key = "app.palette.hint_key_only"
)

func init() {
	register(EN, map[Key]string{
		AppPalGroupExport:   "EXPORT",
		AppPalGroupGeneral:  "GENERAL",
		AppPalGroupExplorer: "EXPLORER",
		AppPalGroupEditor:   "EDITOR · NORMAL",
		AppPalGroupCommands: "COMMANDS",
		AppPalGroupPreview:  "PREVIEW",
		AppPalGroupGraph:    "GRAPH",
		AppPalGroupGlobal:   "GLOBAL GRAPH",
		AppPalGroupMouse:    "MOUSE",
		AppPalShell:         "From the shell",
		AppPalShellLong:     "Run misoku export note.md in a terminal to write the PDF without opening Misoku.",
		AppPalPlaceholder:   "Type to search keys and commands",
		AppPalNoMatch:       "Nothing matches the search.",
		AppPalTitle:         "MISOKU KEYMAPS & COMMANDS",
		AppPalHint:          " Enter run · ↑↓ move · Esc close",
		AppPalHintNeedsNote: " Needs an open note · Esc close",
		AppPalHintKeyOnly:   " Not a command: use the key · Esc close",
	})
	register(ES, map[Key]string{
		AppPalGroupExport:   "EXPORTAR",
		AppPalGroupGeneral:  "GENERAL",
		AppPalGroupExplorer: "EXPLORADOR",
		AppPalGroupEditor:   "EDITOR · NORMAL",
		AppPalGroupCommands: "COMANDOS",
		AppPalGroupPreview:  "VISTA PREVIA",
		AppPalGroupGraph:    "GRAFO",
		AppPalGroupGlobal:   "GRAFO GLOBAL",
		AppPalGroupMouse:    "RATÓN",
		AppPalShell:         "Desde la terminal",
		AppPalShellLong:     "Ejecuta misoku export nota.md en una terminal para escribir el PDF sin abrir Misoku.",
		AppPalPlaceholder:   "Escribe para buscar teclas y comandos",
		AppPalNoMatch:       "Nada coincide con la búsqueda.",
		AppPalTitle:         "ATAJOS Y COMANDOS DE MISOKU",
		AppPalHint:          " Enter ejecuta · ↑↓ mueve · Esc cierra",
		AppPalHintNeedsNote: " Necesita una nota abierta · Esc cierra",
		AppPalHintKeyOnly:   " No es un comando: usa la tecla · Esc cierra",
	})
}

// Recovering the tabs left unsaved when Misoku closed.
const (
	AppRecoveryMkdir      Key = "app.recovery.mkdir"
	AppRecoveryProtectDir Key = "app.recovery.protect_dir"
	AppRecoveryEncode     Key = "app.recovery.encode"
	AppRecoveryCreate     Key = "app.recovery.create"
	AppRecoveryWrite      Key = "app.recovery.write"
	AppRecoveryFlush      Key = "app.recovery.flush"
	AppRecoveryClose      Key = "app.recovery.close"
	AppRecoveryProtect    Key = "app.recovery.protect"
	AppRecoveryKeep       Key = "app.recovery.keep"
	AppRecoveryUnreadable Key = "app.recovery.unreadable"
	AppRecoveryVersion    Key = "app.recovery.version"
	AppRecoveryElsewhere  Key = "app.recovery.elsewhere"
	AppRecoveryReadFailed Key = "app.recovery.read_failed"
	AppRecoveryRemove     Key = "app.recovery.remove"
	AppRecoveryTitle      Key = "app.recovery.title"
	AppRecoveryTabs       Key = "app.recovery.tabs"
	AppRecoveryRestore    Key = "app.recovery.restore"
	AppRecoveryLater      Key = "app.recovery.later"
	AppRecoveryNotRemoved Key = "app.recovery.not_removed"
	AppRecoveryRecovered  Key = "app.recovery.recovered"
	AppRecoveryDiscarded  Key = "app.recovery.discarded"
)

func init() {
	register(EN, map[Key]string{
		AppRecoveryMkdir:      "failed to create the recovery folder",
		AppRecoveryProtectDir: "failed to protect the recovery folder",
		AppRecoveryEncode:     "failed to encode the recovery file",
		AppRecoveryCreate:     "failed to create the recovery file",
		AppRecoveryWrite:      "failed to write the recovery file",
		AppRecoveryFlush:      "failed to flush the recovery file",
		AppRecoveryClose:      "failed to close the recovery file",
		AppRecoveryProtect:    "failed to protect the recovery file",
		AppRecoveryKeep:       "failed to keep the recovery file",
		AppRecoveryUnreadable: "%s is unreadable",
		AppRecoveryVersion:    "%s has version %d",
		AppRecoveryElsewhere:  "Unsaved notes from %s are kept in %s",
		AppRecoveryReadFailed: "Recovery file could not be read: %s",
		AppRecoveryRemove:     "Could not remove the recovery file: %s",
		AppRecoveryTitle:      "Recover unsaved notes?",

		AppRecoveryTabs + one:        "%d tab(s) were not saved when Misoku closed.",
		AppRecoveryTabs + other:      "%d tab(s) were not saved when Misoku closed.",
		AppRecoveryRecovered + one:   "Recovered %d unsaved tab(s)",
		AppRecoveryRecovered + other: "Recovered %d unsaved tab(s)",

		AppRecoveryRestore:    "Restore",
		AppRecoveryLater:      "Later",
		AppRecoveryNotRemoved: "Recovered, but the file could not be removed: %s",
		AppRecoveryDiscarded:  "Recovery discarded",
	})
	register(ES, map[Key]string{
		AppRecoveryMkdir:      "no se pudo crear la carpeta de recuperación",
		AppRecoveryProtectDir: "no se pudo proteger la carpeta de recuperación",
		AppRecoveryEncode:     "no se pudo codificar el archivo de recuperación",
		AppRecoveryCreate:     "no se pudo crear el archivo de recuperación",
		AppRecoveryWrite:      "no se pudo escribir el archivo de recuperación",
		AppRecoveryFlush:      "no se pudo volcar el archivo de recuperación",
		AppRecoveryClose:      "no se pudo cerrar el archivo de recuperación",
		AppRecoveryProtect:    "no se pudo proteger el archivo de recuperación",
		AppRecoveryKeep:       "no se pudo conservar el archivo de recuperación",
		AppRecoveryUnreadable: "%s no se puede leer",
		AppRecoveryVersion:    "%s tiene la versión %d",
		AppRecoveryElsewhere:  "Las notas sin guardar de %s se conservan en %s",
		AppRecoveryReadFailed: "No se pudo leer el archivo de recuperación: %s",
		AppRecoveryRemove:     "No se pudo quitar el archivo de recuperación: %s",
		AppRecoveryTitle:      "¿Recuperar las notas sin guardar?",

		AppRecoveryTabs + one:        "%d pestaña no se guardó cuando Misoku se cerró.",
		AppRecoveryTabs + other:      "%d pestañas no se guardaron cuando Misoku se cerró.",
		AppRecoveryRecovered + one:   "Se recuperó %d pestaña sin guardar",
		AppRecoveryRecovered + other: "Se recuperaron %d pestañas sin guardar",

		AppRecoveryRestore:    "Restaurar",
		AppRecoveryLater:      "Más tarde",
		AppRecoveryNotRemoved: "Se recuperó, pero no se pudo quitar el archivo: %s",
		AppRecoveryDiscarded:  "Recuperación descartada",
	})
}

// The sync setup assistant's own lines (its steps' words come from gitsync).
const (
	AppSetupRunning      Key = "app.setup.running"
	AppSetupCancelling   Key = "app.setup.cancelling"
	AppSetupAlready      Key = "app.setup.already"
	AppSetupNoConfig     Key = "app.setup.no_config"
	AppSetupCancelled    Key = "app.setup.cancelled"
	AppSetupOtherDialog  Key = "app.setup.other_dialog"
	AppSetupCreate       Key = "app.setup.create"
	AppSetupStart        Key = "app.setup.start"
	AppSetupMix          Key = "app.setup.mix"
	AppSetupRepoHint     Key = "app.setup.repo_hint"
	AppSetupQuitHeld     Key = "app.setup.quit_held"
	AppSetupGitHint      Key = "app.setup.git_hint"
	AppSetupGitHubHint   Key = "app.setup.github_hint"
	AppSetupName         Key = "app.setup.name"
	AppSetupEmail        Key = "app.setup.email"
	AppSetupNameLine     Key = "app.setup.name_line"
	AppSetupEmailLine    Key = "app.setup.email_line"
	AppSetupSaveIdentity Key = "app.setup.save_identity"
	AppSetupPrivate      Key = "app.setup.private"
	AppSetupNotAllowed   Key = "app.setup.not_allowed"
	AppSetupDone         Key = "app.setup.done"
)

func init() {
	register(EN, map[Key]string{
		AppSetupRunning:      "Setting up sync… Esc cancels",
		AppSetupCancelling:   "Cancelling sync setup…",
		AppSetupAlready:      "Sync setup is already running…",
		AppSetupNoConfig:     "Sync setup needs Misoku's config file, and there is none here",
		AppSetupCancelled:    "Sync setup cancelled",
		AppSetupOtherDialog:  "Sync setup stopped: another dialog is open",
		AppSetupCreate:       "Create",
		AppSetupStart:        "Start",
		AppSetupMix:          "Mix them",
		AppSetupRepoHint:     "Misoku creates it private on GitHub",
		AppSetupQuitHeld:     "Sync setup is running: Esc cancels it, then quit",
		AppSetupGitHint:      "git writes it in your notes' history",
		AppSetupGitHubHint:   "From your GitHub account; change it if you like",
		AppSetupName:         "Your name, for git",
		AppSetupEmail:        "Your email, for git",
		AppSetupNameLine:     "Name: %s",
		AppSetupEmailLine:    "Email: %s",
		AppSetupSaveIdentity: "Save your git identity",
		AppSetupPrivate:      "%s · private",
		AppSetupNotAllowed:   "%s · %s, not allowed",
		AppSetupDone:         "Sync is set up",
	})
	register(ES, map[Key]string{
		AppSetupRunning:      "Configurando la sincronización… Esc cancela",
		AppSetupCancelling:   "Cancelando la configuración de la sincronización…",
		AppSetupAlready:      "La configuración de la sincronización ya está en marcha…",
		AppSetupNoConfig:     "La configuración de la sincronización necesita el archivo de configuración de Misoku, y aquí no hay ninguno",
		AppSetupCancelled:    "Configuración de la sincronización cancelada",
		AppSetupOtherDialog:  "Configuración de la sincronización detenida: hay otro diálogo abierto",
		AppSetupCreate:       "Crear",
		AppSetupStart:        "Empezar",
		AppSetupMix:          "Mezclarlas",
		AppSetupRepoHint:     "Misoku lo crea privado en GitHub",
		AppSetupQuitHeld:     "La configuración de la sincronización está en marcha: Esc la cancela, luego sal",
		AppSetupGitHint:      "git lo escribe en el historial de tus notas",
		AppSetupGitHubHint:   "De tu cuenta de GitHub; cámbialo si quieres",
		AppSetupName:         "Tu nombre, para git",
		AppSetupEmail:        "Tu correo, para git",
		AppSetupNameLine:     "Nombre: %s",
		AppSetupEmailLine:    "Correo: %s",
		AppSetupSaveIdentity: "Guardar tu identidad de git",
		AppSetupPrivate:      "%s · privado",
		AppSetupNotAllowed:   "%s · %s, no permitido",
		AppSetupDone:         "La sincronización está configurada",
	})
}

// Tags, templates and themes.
const (
	AppIndexing       Key = "app.indexing"
	AppNoTags         Key = "app.no_tags"
	AppTagRow         Key = "app.tag_row"
	AppTagsTitle      Key = "app.tags_title"
	AppNoTemplates    Key = "app.no_templates"
	AppTemplatesTitle Key = "app.templates_title"
	AppTemplateError  Key = "app.template_error"
	AppTemplateDone   Key = "app.template_done"
	AppThemeSaveError Key = "app.theme.save_error"
	AppThemeNotFound  Key = "app.theme.not_found"
	AppThemeSet       Key = "app.theme.set"
	AppThemeFilter    Key = "app.theme.filter"
	AppThemesTitle    Key = "app.theme.title"
	AppThemeDark      Key = "app.theme.dark"
	AppThemeLight     Key = "app.theme.light"
	AppThemeHelp      Key = "app.theme.help"
)

func init() {
	register(EN, map[Key]string{
		AppIndexing: "Indexing notes…",
		AppNoTags:   "No tags yet: write #tag in a note, or tags: in its front matter",

		AppTagRow + one:   "#%s · %d note",
		AppTagRow + other: "#%s · %d notes",

		AppTagsTitle:      "Tags",
		AppNoTemplates:    "No templates yet: create notes in the folder %s/ of the vault",
		AppTemplatesTitle: "Templates",
		AppTemplateError:  "Template %s: %s",
		AppTemplateDone:   "Inserted the template %s: u undoes it",
		AppThemeSaveError: "Could not save theme: %s",
		AppThemeNotFound:  "E185: Cannot find color scheme '%s'",
		AppThemeSet:       "Theme: %s",
		AppThemeFilter:    "Filter themes…",
		AppThemesTitle:    "Themes",
		AppThemeDark:      "dark",
		AppThemeLight:     "light",
		AppThemeHelp:      " ↑↓ preview · Enter apply · Esc revert",
	})
	register(ES, map[Key]string{
		AppIndexing: "Indexando las notas…",
		AppNoTags:   "Aún no hay etiquetas: escribe #etiqueta en una nota, o tags: en su front matter",

		AppTagRow + one:   "#%s · %d nota",
		AppTagRow + other: "#%s · %d notas",

		AppTagsTitle:      "Etiquetas",
		AppNoTemplates:    "Aún no hay plantillas: crea notas en la carpeta %s/ del vault",
		AppTemplatesTitle: "Plantillas",
		AppTemplateError:  "Plantilla %s: %s",
		AppTemplateDone:   "Se insertó la plantilla %s: u lo deshace",
		AppThemeSaveError: "No se pudo guardar el tema: %s",
		AppThemeNotFound:  "E185: No se encuentra el esquema de colores '%s'",
		AppThemeSet:       "Tema: %s",
		AppThemeFilter:    "Filtra los temas…",
		AppThemesTitle:    "Temas",
		AppThemeDark:      "oscuro",
		AppThemeLight:     "claro",
		AppThemeHelp:      " ↑↓ previsualiza · Enter aplica · Esc revierte",
	})
}

// The menus and prompts of the preview's blocks: boards, calendars,
// charts, forms, timelines and properties.
const (
	AppBoardCards        Key = "app.board.cards"
	AppBoardNewCard      Key = "app.board.new_card"
	AppBoardNewCardHint  Key = "app.board.new_card_hint"
	AppBoardMoveTo       Key = "app.board.move_to"
	AppBoardRenameCard   Key = "app.board.rename_card"
	AppBoardCardDeleted  Key = "app.board.card_deleted"
	AppBoardMoved        Key = "app.board.moved"
	AppBoardAddCard      Key = "app.board.add_card"
	AppBoardRenameColumn Key = "app.board.rename_column"
	AppBoardDeleteColumn Key = "app.board.delete_column"
	AppCalNewEvent       Key = "app.cal.new_event"
	AppCalMoveDate       Key = "app.cal.move_date"
	AppCalRenameEvent    Key = "app.cal.rename_event"
	AppCalMoveTo         Key = "app.cal.move_to"
	AppChartAdd          Key = "app.chart.add"
	AppChartAddHint      Key = "app.chart.add_hint"
	AppChartUse          Key = "app.chart.use"
	AppChartEdit         Key = "app.chart.edit"
	AppChartRenameLabel  Key = "app.chart.rename_label"
	AppChartValueFor     Key = "app.chart.value_for"
	AppChartNotNumber    Key = "app.chart.not_number"
	AppChartRename       Key = "app.chart.rename"
	AppFormShort         Key = "app.form.short"
	AppFormLong          Key = "app.form.long"
	AppFormChoice        Key = "app.form.choice"
	AppFormMulti         Key = "app.form.multi"
	AppFormRating        Key = "app.form.rating"
	AppFormDate          Key = "app.form.date"
	AppFormEmail         Key = "app.form.email"
	AppFormTypeOf        Key = "app.form.type_of"
	AppFormOptions3      Key = "app.form.options3"
	AppFormOptionsFor    Key = "app.form.options_for"
	AppFormOptionsHint   Key = "app.form.options_hint"
	AppFormOption1       Key = "app.form.option1"
	AppFormTitle         Key = "app.form.title"
	AppFormDescription   Key = "app.form.description"
	AppFormAnswerHint    Key = "app.form.answer_hint"
	AppFormNewOption     Key = "app.form.new_option"
	AppFormRequired      Key = "app.form.required"
	AppFormOptional      Key = "app.form.optional"
	AppFormChangeType    Key = "app.form.change_type"
	AppFormClearAnswer   Key = "app.form.clear_answer"
	AppFormDeleteQ       Key = "app.form.delete_question"
	AppFormRenameQ       Key = "app.form.rename_question"
	AppFormNewQ          Key = "app.form.new_question"
	AppFormNewQHint      Key = "app.form.new_question_hint"
	AppEditProperty      Key = "app.edit_property"
	AppTlNew             Key = "app.tl.new"
	AppTlNewHint         Key = "app.tl.new_hint"
	AppTlDayLater        Key = "app.tl.day_later"
	AppTlDayEarlier      Key = "app.tl.day_earlier"
	AppTlWeekLater       Key = "app.tl.week_later"
	AppTlWeekEarlier     Key = "app.tl.week_earlier"
	AppTlExtend          Key = "app.tl.extend"
	AppTlShorten         Key = "app.tl.shorten"
	AppTlSetDates        Key = "app.tl.set_dates"
	AppTlStart           Key = "app.tl.start"
	AppTlEnd             Key = "app.tl.end"
	AppPromptHelp        Key = "app.prompt_help"
)

func init() {
	register(EN, map[Key]string{
		AppBoardCards:        "Cards",
		AppBoardNewCard:      "New card in %s",
		AppBoardNewCardHint:  "Lines you add under the card in the editor become its description",
		AppBoardMoveTo:       "Move to → %s",
		AppBoardRenameCard:   "Rename card",
		AppBoardCardDeleted:  "Card deleted",
		AppBoardMoved:        "Moved to %s",
		AppBoardAddCard:      "Add card",
		AppBoardRenameColumn: "Rename column",
		AppBoardDeleteColumn: "Delete column and its cards",
		AppCalNewEvent:       "New event on %s",
		AppCalMoveDate:       "Move to another date",
		AppCalRenameEvent:    "Rename event",
		AppCalMoveTo:         "Move \"%s\" to",
		AppChartAdd:          "Add a value",
		AppChartAddHint:      "Label: number, e.g. May: 18",
		AppChartUse:          "Use Label: number",
		AppChartEdit:         "Edit value",
		AppChartRenameLabel:  "Rename label",
		AppChartValueFor:     "Value for %s",
		AppChartNotNumber:    "Not a number: %s",
		AppChartRename:       "Rename %s",
		AppFormShort:         "Short answer",
		AppFormLong:          "Long answer",
		AppFormChoice:        "Single choice  ○",
		AppFormMulti:         "Multiple choice  ☐",
		AppFormRating:        "Rating  ☆☆☆☆☆",
		AppFormDate:          "Date",
		AppFormEmail:         "Email",
		AppFormTypeOf:        "Type of \"%s\"",
		AppFormOptions3:      "Option 1 | Option 2 | Option 3",
		AppFormOptionsFor:    "Options for \"%s\"",
		AppFormOptionsHint:   "Separate options with | or commas",
		AppFormOption1:       "Option 1",
		AppFormTitle:         "Form title",
		AppFormDescription:   "Description",
		AppFormAnswerHint:    "Your answer (saved in the note as \"= answer\")",
		AppFormNewOption:     "New option for %s",
		AppFormRequired:      "Make required",
		AppFormOptional:      "Make optional",
		AppFormChangeType:    "Change type",
		AppFormClearAnswer:   "Clear answer",
		AppFormDeleteQ:       "Delete question",
		AppFormRenameQ:       "Rename question",
		AppFormNewQ:          "New question",
		AppFormNewQHint:      "Next you'll pick its type (text, choice, rating…)",
		AppEditProperty:      "Edit property",
		AppTlNew:             "New timeline item",
		AppTlNewHint:         "Next you'll pick the start and end dates",
		AppTlDayLater:        "Move 1 day later",
		AppTlDayEarlier:      "Move 1 day earlier",
		AppTlWeekLater:       "Move 1 week later",
		AppTlWeekEarlier:     "Move 1 week earlier",
		AppTlExtend:          "Extend by 1 day",
		AppTlShorten:         "Shorten by 1 day",
		AppTlSetDates:        "Set dates…",
		AppTlStart:           "Start date · %s",
		AppTlEnd:             "End date · %s",
		AppPromptHelp:        " Enter save · Esc cancel",
	})
	register(ES, map[Key]string{
		AppBoardCards:        "Tarjetas",
		AppBoardNewCard:      "Tarjeta nueva en %s",
		AppBoardNewCardHint:  "Las líneas que añadas bajo la tarjeta en el editor serán su descripción",
		AppBoardMoveTo:       "Mover a → %s",
		AppBoardRenameCard:   "Renombrar tarjeta",
		AppBoardCardDeleted:  "Tarjeta eliminada",
		AppBoardMoved:        "Movida a %s",
		AppBoardAddCard:      "Añadir tarjeta",
		AppBoardRenameColumn: "Renombrar columna",
		AppBoardDeleteColumn: "Eliminar la columna y sus tarjetas",
		AppCalNewEvent:       "Evento nuevo el %s",
		AppCalMoveDate:       "Mover a otra fecha",
		AppCalRenameEvent:    "Renombrar evento",
		AppCalMoveTo:         "Mover \"%s\" a",
		AppChartAdd:          "Añadir un valor",
		AppChartAddHint:      "Etiqueta: número, p. ej. Mayo: 18",
		AppChartUse:          "Usa Etiqueta: número",
		AppChartEdit:         "Editar valor",
		AppChartRenameLabel:  "Renombrar etiqueta",
		AppChartValueFor:     "Valor de %s",
		AppChartNotNumber:    "No es un número: %s",
		AppChartRename:       "Renombrar %s",
		AppFormShort:         "Respuesta corta",
		AppFormLong:          "Respuesta larga",
		AppFormChoice:        "Opción única  ○",
		AppFormMulti:         "Opción múltiple  ☐",
		AppFormRating:        "Valoración  ☆☆☆☆☆",
		AppFormDate:          "Fecha",
		AppFormEmail:         "Correo",
		AppFormTypeOf:        "Tipo de \"%s\"",
		AppFormOptions3:      "Opción 1 | Opción 2 | Opción 3",
		AppFormOptionsFor:    "Opciones de \"%s\"",
		AppFormOptionsHint:   "Separa las opciones con | o comas",
		AppFormOption1:       "Opción 1",
		AppFormTitle:         "Título del formulario",
		AppFormDescription:   "Descripción",
		AppFormAnswerHint:    "Tu respuesta (se guarda en la nota como \"= respuesta\")",
		AppFormNewOption:     "Opción nueva para %s",
		AppFormRequired:      "Hacer obligatoria",
		AppFormOptional:      "Hacer opcional",
		AppFormChangeType:    "Cambiar tipo",
		AppFormClearAnswer:   "Borrar respuesta",
		AppFormDeleteQ:       "Eliminar pregunta",
		AppFormRenameQ:       "Renombrar pregunta",
		AppFormNewQ:          "Pregunta nueva",
		AppFormNewQHint:      "Luego elegirás su tipo (texto, opción, valoración…)",
		AppEditProperty:      "Editar propiedad",
		AppTlNew:             "Elemento nuevo de la línea de tiempo",
		AppTlNewHint:         "Luego elegirás las fechas de inicio y fin",
		AppTlDayLater:        "Mover 1 día después",
		AppTlDayEarlier:      "Mover 1 día antes",
		AppTlWeekLater:       "Mover 1 semana después",
		AppTlWeekEarlier:     "Mover 1 semana antes",
		AppTlExtend:          "Alargar 1 día",
		AppTlShorten:         "Acortar 1 día",
		AppTlSetDates:        "Fijar fechas…",
		AppTlStart:           "Fecha de inicio · %s",
		AppTlEnd:             "Fecha de fin · %s",
		AppPromptHelp:        " Enter guarda · Esc cancela",
	})
}

// Odds and ends of the editor's screens.
const (
	AppWrittenNew    Key = "app.written_new"
	AppFolderRefused Key = "app.folder_refused"
	AppPickerHelp    Key = "app.picker_help"
	AppFinderIn      Key = "app.finder.in"
	AppFinderCount   Key = "app.finder.count"
)

func init() {
	register(EN, map[Key]string{
		AppWrittenNew:    "\"%s\" written",
		AppFolderRefused: "%s: %s; using %s",
		AppPickerHelp:    " ←→↑↓ day · [ ] month · t today · Enter",
		AppFinderIn:      "  in %s/",
		AppFinderCount:   "%d results · %d notes ",
	})
	register(ES, map[Key]string{
		AppWrittenNew:    "\"%s\" escrito",
		AppFolderRefused: "%s: %s; se usa %s",
		AppPickerHelp:    " ←→↑↓ día · [ ] mes · t hoy · Enter",
		AppFinderIn:      "  en %s/",
		AppFinderCount:   "%d resultados · %d notas ",
	})
}
