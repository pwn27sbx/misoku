package i18n

// The words of the interface's building blocks: the keymap help, the ex
// commands, the "/" block menu and the editor, the home screen, the
// preview and the explorer.

// KeymapHelp is the one-line help of a keymap action, by the action's
// config name ("new_note", "explorer.up").
func KeymapHelp(action string) Key { return Key("keymap." + action + ".help") }

// KeymapLong is the sentence or two that explains a keymap action.
func KeymapLong(action string) Key { return Key("keymap." + action + ".long") }

// CommandHelp is the one-line help of an ex command, by its id ("w",
// "graph_all").
func CommandHelp(id string) Key { return Key("command." + id + ".help") }

// CommandLong explains an ex command.
func CommandLong(id string) Key { return Key("command." + id + ".long") }

// KeymapGroup is the name of a leader guide group, by its id ("note").
func KeymapGroup(group string) Key { return Key("keymap.group." + group) }

// SlashLabel is the label of a "/" menu item, by its id ("h1", "board").
func SlashLabel(id string) Key { return Key("slash." + id) }

// SlashSection is the header of a "/" menu section, by its id ("basic").
func SlashSection(id string) Key { return Key("slash.section." + id) }

// The keymap's own messages: the name of a leader group with no name of
// its own, and the warnings about the "keys" section of config.json.
const (
	KeymapMoreGroup         Key = "keymap.group.more"
	KeysUnknownAction       Key = "keymap.warn.unknown_action"
	KeysSeveralNeedSpace    Key = "keymap.warn.several_need_space"
	KeysOneKey              Key = "keymap.warn.one_key"
	KeysBoundTwice          Key = "keymap.warn.bound_twice"
	KeysPrefixClash         Key = "keymap.warn.prefix_clash"
	KeysLeaderInExplorer    Key = "keymap.warn.leader_in_explorer"
	KeysShadowedOnExplorer  Key = "keymap.warn.shadowed_on_explorer"
	EditorNotACommand       Key = "editor.not_a_command"
	EditorTrailing          Key = "editor.trailing"
	EditorTrailingGraph     Key = "editor.trailing.graph"
	EditorTrailingSync      Key = "editor.trailing.sync"
	EditorBadStatus         Key = "editor.bad_status"
	EditorNotInTable        Key = "editor.not_in_table"
	EditorClipboardEmpty    Key = "editor.clipboard_empty"
	EditorPasteNeeds        Key = "editor.paste_needs"
	EditorNoNote            Key = "editor.no_note"
	EditorHintNewNote       Key = "editor.hint.new_note"
	EditorHintSearch        Key = "editor.hint.search"
	EditorHintExplorer      Key = "editor.hint.explorer"
	EditorPlaceholder       Key = "editor.placeholder"
	EditorNoMatches         Key = "editor.no_matches"
	EditorNoResults         Key = "editor.no_results"
	EditorNoPreviousSearch  Key = "editor.search.no_previous"
	EditorPatternNotFound   Key = "editor.search.not_found"
	EditorSearchWrappedTop  Key = "editor.search.wrapped_top"
	EditorSearchWrappedBot  Key = "editor.search.wrapped_bottom"
	EditorUndoTooLarge      Key = "editor.undo.too_large"
	EditorUndoOldest        Key = "editor.undo.oldest"
	EditorUndoNewest        Key = "editor.undo.newest"
	EditorUndoBefore        Key = "editor.undo.before"
	EditorUndoAfter         Key = "editor.undo.after"
	EditorNewColumn         Key = "editor.table.new_column"
	SlashColumnN            Key = "slash.starter.column_n"
	SlashBoardStart         Key = "slash.starter.board.before"
	SlashBoardRest          Key = "slash.starter.board.after"
	SlashTimelineRest       Key = "slash.starter.timeline.after"
	SlashChartRest          Key = "slash.starter.chart.after"
	SlashFormRest           Key = "slash.starter.form.after"
	SlashFlowStart          Key = "slash.starter.flow.before"
	SlashFlowRest           Key = "slash.starter.flow.after"
	SlashHintClipboard      Key = "slash.hint.clipboard"
	DashNewNote             Key = "dashboard.new_note"
	DashFindNote            Key = "dashboard.find_note"
	DashOpenExplorer        Key = "dashboard.open_explorer"
	DashThemes              Key = "dashboard.themes"
	DashKeymaps             Key = "dashboard.keymaps"
	DashGraph               Key = "dashboard.graph"
	DashSettings            Key = "dashboard.settings"
	DashQuit                Key = "dashboard.quit"
	DashSync                Key = "dashboard.sync"
	DashTagline             Key = "dashboard.tagline"
	DashRecent              Key = "dashboard.recent"
	DashFolders             Key = "dashboard.folders"
	DashNotes               Key = "dashboard.notes"
	DashFooterHint          Key = "dashboard.footer_hint"
	PreviewTop              Key = "preview.top"
	PreviewBottom           Key = "preview.bottom"
	PreviewNothing          Key = "preview.nothing"
	PreviewRenderError      Key = "preview.render_error"
	PreviewHiddenLines      Key = "preview.hidden_lines"
	PreviewCodeLabel        Key = "preview.code.label"
	PreviewCodeInfo         Key = "preview.code.info"
	PreviewAddRow           Key = "preview.table.add_row"
	PreviewAddColumn        Key = "preview.table.add_column"
	PreviewImage            Key = "preview.image"
	PreviewNotFound         Key = "preview.not_found"
	PreviewNoSuchNote       Key = "preview.no_such_note"
	PreviewSectionNotFound  Key = "preview.section_not_found"
	PreviewMentionButton    Key = "preview.mention_button"
	PreviewUnlinkedMentions Key = "preview.unlinked_mentions"
	PreviewMentions         Key = "preview.mentions"
	PreviewLinkedFrom       Key = "preview.linked_from"
	PreviewBoardCards       Key = "preview.board.cards"
	PreviewBoardEmpty       Key = "preview.board.empty"
	PreviewNewPage          Key = "preview.board.new_page"
	PreviewCalendarEvents   Key = "preview.calendar.events"
	PreviewCalendarMore     Key = "preview.calendar.more"
	PreviewToday            Key = "preview.calendar.today"
	PreviewWeekdays         Key = "preview.calendar.weekdays"
	PreviewMonths           Key = "preview.months"
	PreviewMonthsShort      Key = "preview.months.short"
	PreviewChartEmpty       Key = "preview.chart.empty"
	PreviewAddValue         Key = "preview.chart.add_value"
	PreviewTimelineEmpty    Key = "preview.timeline.empty"
	PreviewTimelineNew      Key = "preview.timeline.new"
	PreviewFormTitle        Key = "preview.form.title"
	PreviewFormDescription  Key = "preview.form.description"
	PreviewFormUpTo         Key = "preview.form.up_to"
	PreviewFormAddOption    Key = "preview.form.add_option"
	PreviewFormDate         Key = "preview.form.date"
	PreviewFormLong         Key = "preview.form.long"
	PreviewFormEmail        Key = "preview.form.email"
	PreviewFormAnswer       Key = "preview.form.answer"
	PreviewFormNoQuestions  Key = "preview.form.no_questions"
	PreviewFormAddQuestion  Key = "preview.form.add_question"
	PreviewFlowEmpty        Key = "preview.flow.empty"
	PreviewMathEmpty        Key = "preview.math.empty"
	PreviewImageMissing     Key = "preview.image.missing"
	PreviewImageUnsupported Key = "preview.image.unsupported"
	PreviewImageEmpty       Key = "preview.image.empty"
	PreviewImageRemote      Key = "preview.image.remote"
	PreviewImageNoFolder    Key = "preview.image.no_folder"
	SidebarFindNote         Key = "sidebar.find_note"
	SidebarEmpty            Key = "sidebar.empty"
	SidebarEditTitle        Key = "sidebar.edit.title"
	SidebarEditHint         Key = "sidebar.edit.hint"
	SidebarEditEmpty        Key = "sidebar.edit.empty"
	SidebarNamePlaceholder  Key = "sidebar.name_placeholder"
)

// keymapTexts holds each action's help and explanation, English then
// Spanish.
var keymapTexts = []struct{ action, enHelp, enLong, esHelp, esLong string }{
	{"new_note", "New note (unsaved tab)",
		"Opens an empty tab for a new note. It is written to disk on the first save, which asks for its folder.",
		"Nueva nota (pestaña sin guardar)",
		"Abre una pestaña vacía para una nota nueva. Se escribe en el disco al guardarla por primera vez, y entonces pregunta su carpeta."},
	{"save", "Save (new notes: pick folder)",
		"Writes the note in this tab to disk. A note that was never saved asks which folder it goes in.",
		"Guardar (notas nuevas: elegir carpeta)",
		"Escribe en el disco la nota de esta pestaña. Una nota que nunca se guardó pregunta en qué carpeta va."},
	{"find_note", "Find note (opens in this tab)",
		"Searches the names and the text of every note in the vault; the note you pick opens in this tab.",
		"Buscar nota (se abre en esta pestaña)",
		"Busca en los nombres y el texto de todas las notas del vault; la nota que elijas se abre en esta pestaña."},
	{"toggle_explorer", "Toggle explorer",
		"Shows or hides the explorer, the tree of folders and notes on the left. On a narrow window it opens over the editor.",
		"Mostrar u ocultar el explorador",
		"Muestra u oculta el explorador, el árbol de carpetas y notas de la izquierda. En una ventana estrecha se abre encima del editor."},
	{"focus_explorer", "Focus / close explorer",
		"Focuses the explorer, showing it if it was hidden, so its own keys work; when it already has the focus it closes and gives the focus back to the editor. With no note open it keeps the explorer, as there is nothing to go back to.",
		"Ir al explorador / cerrarlo",
		"Lleva el foco al explorador, y lo muestra si estaba oculto, para que funcionen sus teclas; si ya tiene el foco, se cierra y devuelve el foco al editor. Sin ninguna nota abierta se queda en el explorador, porque no hay adónde volver."},
	{"next_pane", "Next pane",
		"Moves the focus to the next pane: explorer, editor, preview.",
		"Panel siguiente",
		"Lleva el foco al panel siguiente: explorador, editor, preview."},
	{"prev_pane", "Previous pane",
		"Moves the focus to the previous pane: preview, editor, explorer.",
		"Panel anterior",
		"Lleva el foco al panel anterior: preview, editor, explorador."},
	{"pane_left", "Pane to the left",
		"Moves the focus to the pane on the left of this one, and stops at the first.",
		"Panel de la izquierda",
		"Lleva el foco al panel a la izquierda de este, y se detiene en el primero."},
	{"pane_right", "Pane to the right",
		"Moves the focus to the pane on the right of this one, and stops at the last.",
		"Panel de la derecha",
		"Lleva el foco al panel a la derecha de este, y se detiene en el último."},
	{"next_tab", "Next tab",
		"Shows the next open note in the tab bar, wrapping around after the last.",
		"Pestaña siguiente",
		"Muestra la siguiente nota abierta en la barra de pestañas; después de la última vuelve a la primera."},
	{"prev_tab", "Previous tab",
		"Shows the previous open note in the tab bar, wrapping around before the first.",
		"Pestaña anterior",
		"Muestra la nota abierta anterior en la barra de pestañas; antes de la primera pasa a la última."},
	{"go_back", "Back to the previous note of this tab",
		"Returns this tab to the note it showed before gd, the finder or :e replaced it, at the line the cursor was on. Press it again to go further back.",
		"Volver a la nota anterior de esta pestaña",
		"Devuelve esta pestaña a la nota que mostraba antes de que gd, el buscador o :e la reemplazaran, en la línea donde estaba el cursor. Púlsala otra vez para seguir retrocediendo."},
	{"close_tab", "Close tab",
		"Closes the tab of this note. With unsaved changes it asks whether to save them first.",
		"Cerrar pestaña",
		"Cierra la pestaña de esta nota. Si tiene cambios sin guardar, pregunta antes si quieres guardarlos."},
	{"reading_view", "Reading view: rendered and editable, full width",
		"Shows only the note at full width, without line numbers, drawn as the side preview draws it. It stays editable: in NORMAL every block stays rendered, the cursor's block under the cursor-line band, j/k move block by block and dd, yy, o, O, p, P, >> and << act on the whole block while x does nothing there; INSERT shows the cursor's block as Markdown and a selection shows the blocks it touches; every editing key works, and what is clickable in the preview works here too. The same key goes back to the split; the editor view key shows all the Markdown with line numbers.",
		"Vista de lectura: renderizada y editable, a todo el ancho",
		"Muestra solo la nota a todo el ancho, sin números de línea, dibujada como la dibuja el preview. Sigue siendo editable: en NORMAL todos los bloques siguen renderizados, el bloque del cursor bajo la franja de la línea del cursor, j/k se mueven de bloque en bloque y dd, yy, o, O, p, P, >> y << actúan sobre el bloque entero, mientras que x no hace nada ahí; INSERT muestra el bloque del cursor como Markdown y una selección muestra los bloques que toca; todas las teclas de edición funcionan, y lo que se puede pulsar en el preview funciona aquí también. La misma tecla vuelve a la vista dividida; la tecla de la vista de editor muestra todo el Markdown con números de línea."},
	{"editor_view", "Editor view: Markdown full width",
		"Shows only the Markdown editor at full width, with line numbers and without the side preview; the note edits as usual. The same key goes back to the split.",
		"Vista de editor: Markdown a todo el ancho",
		"Muestra solo el editor de Markdown a todo el ancho, con números de línea y sin el preview al lado; la nota se edita como siempre. La misma tecla vuelve a la vista dividida."},
	{"export_pdf", "Export note as PDF",
		"Writes this note as a PDF file in the note's folder, with its blocks rendered.",
		"Exportar la nota como PDF",
		"Escribe esta nota como archivo PDF en la carpeta de la nota, con sus bloques renderizados."},
	{"themes", "Themes",
		"Opens the theme picker: the highlighted theme applies at once, Enter keeps it and Esc goes back.",
		"Temas",
		"Abre el selector de temas: el tema resaltado se aplica al momento, Enter lo deja y Esc vuelve atrás."},
	{"graph", "Graph of this note's links",
		"Shows a map of the notes this note links to and the ones linking to it; move with h j k l and open one with Enter.",
		"Grafo de los enlaces de esta nota",
		"Muestra un mapa de las notas a las que enlaza esta nota y de las que la enlazan; muévete con h j k l y abre una con Enter."},
	{"global_graph", "Graph of every note",
		"Shows a map of every note in the vault, grouped by top-level folder, with the links between them; h j k l move to the nearest note, n and N go round the selected note's links.",
		"Grafo de todas las notas",
		"Muestra un mapa de todas las notas del vault, agrupadas por carpeta principal, con los enlaces entre ellas; h j k l van a la nota más cercana, n y N recorren los enlaces de la nota elegida."},
	{"home", "Home screen",
		"Goes back to the home screen, with recent notes and folders. Open tabs stay open.",
		"Pantalla de inicio",
		"Vuelve a la pantalla de inicio, con las notas y carpetas recientes. Las pestañas abiertas siguen abiertas."},
	{"daily_note", "Today's daily note",
		"Opens today's note, Diario/YYYY-MM-DD.md (daily_dir in config.json names another folder). When it does not exist yet, a new note with that name starts in that folder; nothing is written until you save it. :daily does the same.",
		"Nota diaria de hoy",
		"Abre la nota de hoy, Diario/AAAA-MM-DD.md (daily_dir en config.json nombra otra carpeta). Si aún no existe, empieza una nota nueva con ese nombre en esa carpeta; no se escribe nada hasta que la guardes. :daily hace lo mismo."},
	{"sync_now", "Sync the vault with git now",
		"Syncs the vault with its git remote right away, instead of waiting for the next automatic sync.",
		"Sincronizar el vault con git ahora",
		"Sincroniza el vault con su remoto de git en este momento, sin esperar a la próxima sincronización automática."},
	{"paste_image", "Paste an image from the clipboard",
		"Saves the picture in the clipboard next to this note and links it on a line of its own. In insert mode ctrl+v does the same, and pastes text when the clipboard holds no picture.",
		"Pegar una imagen del portapapeles",
		"Guarda la imagen del portapapeles junto a esta nota y la enlaza en una línea propia. En modo INSERT, ctrl+v hace lo mismo, y pega texto si el portapapeles no tiene una imagen."},
	{"toggle_checkbox", "Toggle checkbox",
		"Checks or unchecks the box of this line, or of every line of the selection: [ ] becomes [x] and [x] or [-] becomes [ ]; a line without a box gets one.",
		"Marcar o desmarcar casilla",
		"Marca o desmarca la casilla de esta línea, o de cada línea de la selección: [ ] pasa a [x] y [x] o [-] pasa a [ ]; una línea sin casilla recibe una."},
	{"remove_checkbox", "Remove checkbox",
		"Takes the box off this line, or off every line of the selection, and keeps the list item: - [x] task becomes - task.",
		"Quitar casilla",
		"Quita la casilla de esta línea, o de cada línea de la selección, y deja el elemento de la lista: - [x] tarea pasa a - tarea."},
	{"next_link", "Next [[link]]",
		"Moves the cursor to the start of the next [[link]] of this note, past the last back to the first; links in code are skipped.",
		"[[Enlace]] siguiente",
		"Lleva el cursor al inicio del siguiente [[enlace]] de esta nota; después del último vuelve al primero; los enlaces dentro de código se saltan."},
	{"prev_link", "Previous [[link]]",
		"Moves the cursor to the start of the previous [[link]] of this note, before the first round to the last; links in code are skipped.",
		"[[Enlace]] anterior",
		"Lleva el cursor al inicio del [[enlace]] anterior de esta nota; antes del primero pasa al último; los enlaces dentro de código se saltan."},
	{"help", "This help",
		"Opens this list of every key and command. Type to filter it; Enter runs a command, Esc closes.",
		"Esta ayuda",
		"Abre esta lista de todas las teclas y comandos. Escribe para filtrarla; Enter ejecuta un comando, Esc la cierra."},
	{"quit", "Quit (asks to save)",
		"Closes Misoku. If a note has unsaved changes, it asks whether to save them first.",
		"Salir (pregunta si guardar)",
		"Cierra Misoku. Si una nota tiene cambios sin guardar, pregunta antes si quieres guardarlos."},

	{"explorer.up", "Move up",
		"In the explorer, moves the selection one row up.",
		"Subir",
		"En el explorador, sube la selección una fila."},
	{"explorer.down", "Move down",
		"In the explorer, moves the selection one row down.",
		"Bajar",
		"En el explorador, baja la selección una fila."},
	{"explorer.top", "First item",
		"In the explorer, jumps to the first row of the tree.",
		"Primer elemento",
		"En el explorador, salta a la primera fila del árbol."},
	{"explorer.bottom", "Last item",
		"In the explorer, jumps to the last row of the tree.",
		"Último elemento",
		"En el explorador, salta a la última fila del árbol."},
	{"explorer.page_down", "Half page down",
		"In the explorer, moves the selection half a screen down.",
		"Media página abajo",
		"En el explorador, baja la selección media pantalla."},
	{"explorer.page_up", "Half page up",
		"In the explorer, moves the selection half a screen up.",
		"Media página arriba",
		"En el explorador, sube la selección media pantalla."},
	{"explorer.open", "Open note / expand",
		"In the explorer, opens the selected note in the editor, or opens and closes the selected folder without opening any note.",
		"Abrir nota / desplegar",
		"En el explorador, abre la nota elegida en el editor, o abre y cierra la carpeta elegida sin abrir ninguna nota."},
	{"explorer.preview", "Preview note (folder: open)",
		"In the explorer, shows the selected note without leaving the explorer; on a folder, opens it.",
		"Ver la nota (carpeta: abrir)",
		"En el explorador, muestra la nota elegida sin salir del explorador; en una carpeta, la abre."},
	{"explorer.expand", "Expand / open",
		"In the explorer, opens the selected folder, or the selected note.",
		"Desplegar / abrir",
		"En el explorador, abre la carpeta elegida, o la nota elegida."},
	{"explorer.collapse", "Collapse / go to parent",
		"In the explorer, closes the selected folder, or moves to the folder that holds the selection.",
		"Plegar / ir a la carpeta superior",
		"En el explorador, cierra la carpeta elegida, o va a la carpeta que contiene la selección."},
	{"explorer.toggle_fold", "Fold / unfold",
		"In the explorer, opens the selected folder if it is closed and closes it if it is open.",
		"Plegar / desplegar",
		"En el explorador, abre la carpeta elegida si está cerrada y la cierra si está abierta."},
	{"explorer.collapse_all", "Collapse all",
		"In the explorer, closes every folder of the tree.",
		"Plegar todo",
		"En el explorador, cierra todas las carpetas del árbol."},
	{"explorer.new_note", "New note or folder (end with / for a folder)",
		"In the explorer, asks for a name in the folder of the selection: test makes a note, test/ a folder and ideas/test both, after confirming. An empty name starts an Untitled note.",
		"Nueva nota o carpeta (termina en / para una carpeta)",
		"En el explorador, pide un nombre en la carpeta de la selección: prueba crea una nota, prueba/ una carpeta e ideas/prueba las dos, después de confirmar. Un nombre vacío empieza una nota Untitled."},
	{"explorer.new_sub_note", "New sub-note",
		"In the explorer, starts a new note nested under the selected one.",
		"Nueva subnota",
		"En el explorador, empieza una nota nueva dentro de la elegida."},
	{"explorer.rename", "Rename",
		"In the explorer, renames the selected note or folder; type the new name and press Enter.",
		"Renombrar",
		"En el explorador, renombra la nota o carpeta elegida; escribe el nombre nuevo y pulsa Enter."},
	{"explorer.delete", "Delete (asks first)",
		"In the explorer, deletes the selected note or folder after asking to confirm.",
		"Borrar (pregunta antes)",
		"En el explorador, borra la nota o carpeta elegida después de pedir confirmación."},
	{"explorer.find", "Find note",
		"In the explorer, opens the note finder, the same one as Find note.",
		"Buscar nota",
		"En el explorador, abre el buscador de notas, el mismo que Buscar nota."},
	{"explorer.file_manager", "Open the vault in Neovim's mini.files",
		"In the explorer, hands the screen to Neovim's mini.files on the vault folder; closing it (q) returns to Misoku, which re-reads the folder at once.",
		"Abrir el vault en mini.files de Neovim",
		"En el explorador, cede la pantalla a mini.files de Neovim sobre la carpeta del vault; al cerrarlo (q) vuelves a Misoku, que vuelve a leer la carpeta al momento."},
	{"explorer.edit_folder", "Edit the folder like text (Oil)",
		"In the explorer, shows the selected folder as editable lines, like Oil: type a line to create with o/O (a trailing / makes a folder), edit a name to rename, dd or erasing the whole line with Backspace to delete, dd then p in another folder (h/l) to move, u to undo. Enter while typing only opens a new line below; nothing touches the disk until you confirm. With edits pending, Enter, =, :w or Ctrl+S (also while typing) lists the actions and applies them after Yes; No returns to the lines with the edits kept. The editor stays open on the same folder, re-read from disk, with the cursor on the new entry. With nothing pending, Enter enters a folder or opens a note. Esc or q leaves; with edits it shows the same confirm: Yes applies and closes, No drops them, Cancel keeps editing.",
		"Editar la carpeta como texto (Oil)",
		"En el explorador, muestra la carpeta elegida como líneas editables, como Oil: escribe una línea con o/O para crear (una / al final crea una carpeta), edita un nombre para renombrar, dd o borrar la línea entera con Backspace para eliminar, dd y luego p en otra carpeta (h/l) para mover, u para deshacer. Enter mientras escribes solo abre una línea nueva debajo; nada toca el disco hasta que confirmes. Con cambios pendientes, Enter, =, :w o Ctrl+S (también mientras escribes) lista las acciones y las aplica tras Sí; No vuelve a las líneas con los cambios intactos. El editor sigue abierto en la misma carpeta, releída del disco, con el cursor en la entrada nueva. Sin nada pendiente, Enter entra en una carpeta o abre una nota. Esc o q sale; con cambios muestra la misma confirmación: Sí aplica y cierra, No los descarta, Cancelar sigue editando."},
	{"explorer.insert", "Start typing in the folder editor",
		"In the explorer, opens the folder editor (m) on the folder holding the selection, with the cursor on it, and passes it the key, as mini.files does: i/I and a/A edit the selected name, o/O type a new line below or above it.",
		"Empezar a escribir en el editor de carpetas",
		"En el explorador, abre el editor de carpetas (m) en la carpeta de la selección, con el cursor en ella, y le pasa la tecla, como hace mini.files: i/I y a/A editan el nombre elegido, o/O escriben una línea nueva debajo o encima."},
}

// commandTexts holds each ex command's help and explanation, English then
// Spanish. Command names, their arguments' placeholders (name, file,
// value) and status values stay as typed.
var commandTexts = []struct{ id, enHelp, enLong, esHelp, esLong string }{
	{"w", "Save",
		"Writes the note to disk. A note that was never saved asks which folder it goes in.",
		"Guardar",
		"Escribe la nota en el disco. Una nota que nunca se guardó pregunta en qué carpeta va."},
	{"saveas", "Save as name.md",
		"Saves the note under a new name, name.md; a path such as folder/name puts it in that folder of the vault. :sav and :saveas do the same.",
		"Guardar como name.md",
		"Guarda la nota con un nombre nuevo, name.md; una ruta como carpeta/name la pone en esa carpeta del vault. :sav y :saveas hacen lo mismo."},
	{"wq", "Save and quit",
		"Saves this note, then closes Misoku. Another note with unsaved changes still asks first.",
		"Guardar y salir",
		"Guarda esta nota y luego cierra Misoku. Si otra nota tiene cambios sin guardar, pregunta antes."},
	{"q", "Quit / discard",
		"Closes Misoku. :q asks before losing unsaved changes; :q! throws them away.",
		"Salir / descartar",
		"Cierra Misoku. :q pregunta antes de perder cambios sin guardar; :q! los descarta."},
	{"bd", "Close tab",
		"Closes this tab. With unsaved changes it asks first; :bd! closes it without saving.",
		"Cerrar pestaña",
		"Cierra esta pestaña. Si tiene cambios sin guardar, pregunta antes; :bd! la cierra sin guardar."},
	{"enew", "New note",
		"Opens an empty tab for a new note, as the new note key does. :e and :edit without a name do the same; with a name they open that note.",
		"Nueva nota",
		"Abre una pestaña vacía para una nota nueva, como la tecla de nueva nota. :e y :edit sin nombre hacen lo mismo; con un nombre abren esa nota."},
	{"e", "Open or start note name",
		"Opens the note name, resolved as a [[link]] written in this note would be, so Projects/AWS works too. A note that does not exist yet starts unsaved with that name; nothing is written until you save it. Bare :e opens an empty tab.",
		"Abrir o empezar la nota name",
		"Abre la nota name, resuelta como lo sería un [[enlace]] escrito en esta nota, así que Proyectos/AWS también funciona. Una nota que aún no existe empieza sin guardar con ese nombre; no se escribe nada hasta que la guardes. :e solo abre una pestaña vacía."},
	{"export", "Export to a file",
		"Writes this note as a PDF. Without a file it goes in the note's folder, named after the note; a file or folder picks where. :pdf does the same.",
		"Exportar a un archivo",
		"Escribe esta nota como PDF. Sin archivo va a la carpeta de la nota, con el nombre de la nota; un archivo o una carpeta eligen dónde. :pdf hace lo mismo."},
	{"graph", "Graph of this note's links",
		"Shows the map of the notes this note links to and the ones linking to it.",
		"Grafo de los enlaces de esta nota",
		"Muestra el mapa de las notas a las que enlaza esta nota y de las que la enlazan."},
	{"graph_all", "Graph of every note",
		"Shows the map of every note in the vault and the links between them, notes without links and links to missing notes included.",
		"Grafo de todas las notas",
		"Muestra el mapa de todas las notas del vault y los enlaces entre ellas, incluidas las notas sin enlaces y los enlaces a notas que no existen."},
	{"daily", "Today's daily note",
		"Opens today's note, Diario/YYYY-MM-DD.md or the daily_dir of config.json. When it does not exist yet, a new note with that name starts in that folder; nothing is written until you save it.",
		"Nota diaria de hoy",
		"Abre la nota de hoy, Diario/AAAA-MM-DD.md o el daily_dir de config.json. Si aún no existe, empieza una nota nueva con ese nombre en esa carpeta; no se escribe nada hasta que la guardes."},
	{"history", "Older versions of this note",
		"Lists the commits of git that changed this note, renames included, and shows the selected version. Enter puts it back in the editor as one change: u undoes it, and nothing is written until you save.",
		"Versiones anteriores de esta nota",
		"Lista los commits de git que cambiaron esta nota, renombres incluidos, y muestra la versión elegida. Enter la devuelve al editor como un solo cambio: u lo deshace, y no se escribe nada hasta que guardes."},
	{"tags", "Every tag, to filter by one",
		"Lists every tag of the vault, #tag in the text or tags: in the front matter, with how many notes carry it. Picking one opens the finder with tag:name, which also finds its nested tags (name/sub).",
		"Todas las etiquetas, para filtrar por una",
		"Lista todas las etiquetas del vault, #etiqueta en el texto o tags: en el front matter, con cuántas notas la llevan. Al elegir una se abre el buscador con tag:nombre, que también encuentra sus etiquetas anidadas (nombre/sub)."},
	{"status", "Notes with a status",
		"Opens the finder with status:value: active, onhold, completed or dropped, the status: of the front matter. Without a value it opens with status: and lists the four.",
		"Notas con un estado",
		"Abre el buscador con status:valor: active, onhold, completed o dropped, el status: del front matter. Sin valor se abre con status: y lista los cuatro."},
	{"messages", "Last messages of the bar",
		"Lists the last 100 messages the status bar showed, with their time, so an error that went away can be read again.",
		"Últimos mensajes de la barra",
		"Lista los últimos 100 mensajes que mostró la barra de estado, con su hora, para volver a leer un error que ya se fue."},
	{"colo", "Switch theme",
		"With a name, applies and saves that theme; without one, opens the theme picker.",
		"Cambiar de tema",
		"Con un nombre, aplica y guarda ese tema; sin nombre, abre el selector de temas."},
	{"settings", "Settings menu",
		"Opens the settings menu: accordion folders, the explorer's hide width, heading icons and bullets, the dialog border, the transparent background and heading numbers. A change applies and is saved at once.",
		"Menú de ajustes",
		"Abre el menú de ajustes: carpetas en acordeón, el ancho para ocultar el explorador, iconos y viñetas de los títulos, el borde de los diálogos, el fondo transparente y la numeración de los títulos. Un cambio se aplica y se guarda al momento."},
	{"sync", "Sync the vault with git",
		"Syncs the vault with its git remote now, like the sync key.",
		"Sincronizar el vault con git",
		"Sincroniza el vault con su remoto de git ahora, como la tecla de sincronizar."},
	{"sync_status", "Sync state, conflict copies",
		"Shows the state of the sync and any conflict copies left to look at.",
		"Estado de la sincronización, copias en conflicto",
		"Muestra el estado de la sincronización y las copias en conflicto que quedan por revisar."},
	{"sync_setup", "Connect the vault to GitHub",
		"Walks through connecting this vault to a GitHub repository, so it syncs on its own.",
		"Conectar el vault a GitHub",
		"Te guía para conectar este vault a un repositorio de GitHub, para que se sincronice solo."},
	{"addrow", "Table row",
		"With the cursor in a Markdown table, adds a row below the cursor. :tr is the short form.",
		"Fila de tabla",
		"Con el cursor en una tabla de Markdown, añade una fila debajo del cursor. :tr es la forma corta."},
	{"addcol", "Table column",
		"With the cursor in a Markdown table, adds a column after the cursor. :tc is the short form.",
		"Columna de tabla",
		"Con el cursor en una tabla de Markdown, añade una columna después del cursor. :tc es la forma corta."},
	{"delrow", "Delete table row / column",
		"With the cursor in a Markdown table, deletes the row or the column under the cursor.",
		"Borrar fila / columna de tabla",
		"Con el cursor en una tabla de Markdown, borra la fila o la columna bajo el cursor."},
	{"tablefmt", "Align the table",
		"With the cursor in a Markdown table, lines up its columns. :tf is the short form.",
		"Alinear la tabla",
		"Con el cursor en una tabla de Markdown, alinea sus columnas. :tf es la forma corta."},
}

// slashTexts holds the "/" menu's labels and section headers, English then
// Spanish. The table operations use the table menu's own keys.
var slashTexts = []struct {
	key    Key
	en, es string
}{
	{SlashSection("table"), "Table", "Tabla"},
	{SlashSection("basic"), "Basic blocks", "Bloques básicos"},
	{SlashSection("views"), "Views", "Vistas"},
	{SlashSection("media"), "Media", "Multimedia"},
	{SlashLabel("format"), "Format table", "Formatear tabla"},
	{SlashLabel("h1"), "Heading 1", "Título 1"},
	{SlashLabel("h2"), "Heading 2", "Título 2"},
	{SlashLabel("h3"), "Heading 3", "Título 3"},
	{SlashLabel("h4"), "Heading 4", "Título 4"},
	{SlashLabel("bullet"), "Bulleted list", "Lista con viñetas"},
	{SlashLabel("numbered"), "Numbered list", "Lista numerada"},
	{SlashLabel("todo"), "To-do list", "Lista de tareas"},
	{SlashLabel("toggle"), "Toggle list", "Lista desplegable"},
	{SlashLabel("page"), "Page", "Página"},
	{SlashLabel("callout"), "Callout", "Destacado"},
	{SlashLabel("quote"), "Quote", "Cita"},
	{SlashLabel("table"), "Table", "Tabla"},
	{SlashLabel("equation"), "Equation", "Ecuación"},
	{SlashLabel("divider"), "Divider", "Separador"},
	{SlashLabel("columns2"), "2 columns", "2 columnas"},
	{SlashLabel("columns3"), "3 columns", "3 columnas"},
	{SlashLabel("columns4"), "4 columns", "4 columnas"},
	{SlashLabel("link"), "Link to page", "Enlace a página"},
	{SlashLabel("template"), "Template", "Plantilla"},
	{SlashLabel("board"), "Board", "Tablero"},
	{SlashLabel("calendar"), "Calendar", "Calendario"},
	{SlashLabel("timeline"), "Timeline", "Cronograma"},
	{SlashLabel("chart"), "Chart", "Gráfico"},
	{SlashLabel("form"), "Form", "Formulario"},
	{SlashLabel("flowchart"), "Flowchart", "Diagrama de flujo"},
	{SlashLabel("cover"), "Cover", "Portada"},
	{SlashLabel("image"), "Image", "Imagen"},
	{SlashLabel("paste"), "Paste image", "Pegar imagen"},
	{SlashLabel("video"), "Video", "Video"},
	{SlashLabel("audio"), "Audio", "Audio"},
	{SlashLabel("file"), "File", "Archivo"},
	{SlashLabel("code"), "Code", "Código"},
}

func init() {
	en, es := map[Key]string{}, map[Key]string{}
	for _, t := range keymapTexts {
		en[KeymapHelp(t.action)], en[KeymapLong(t.action)] = t.enHelp, t.enLong
		es[KeymapHelp(t.action)], es[KeymapLong(t.action)] = t.esHelp, t.esLong
	}
	for _, t := range commandTexts {
		en[CommandHelp(t.id)], en[CommandLong(t.id)] = t.enHelp, t.enLong
		es[CommandHelp(t.id)], es[CommandLong(t.id)] = t.esHelp, t.esLong
	}
	for _, t := range slashTexts {
		en[t.key], es[t.key] = t.en, t.es
	}
	register(EN, en)
	register(ES, es)

	register(EN, map[Key]string{
		KeymapMoreGroup:        "more",
		KeymapGroup("note"):    "note",
		KeymapGroup("view"):    "view",
		KeymapGroup("go"):      "go",
		KeysUnknownAction:      "keys: unknown action %q",
		KeysSeveralNeedSpace:   "%q: a shortcut of several keys must start with space",
		KeysOneKey:             "%q: use one key, or space plus one or two keys",
		KeysBoundTwice:         "keys: %q is bound to both %s and %s; unbind one of them",
		KeysPrefixClash:        "keys: %q on %s is the start of %q on %s, so %s can never be typed; unbind one of them",
		KeysLeaderInExplorer:   "keys: %q on %s starts a leader sequence, so the explorer never gets it; pick another key",
		KeysShadowedOnExplorer: "keys: %q is bound to both %s and %s, and %s runs first even in the explorer; unbind one of them",

		EditorNotACommand:      "E492: Not an editor command: %s",
		EditorTrailing:         "E488: Trailing characters: %s (use %s)",
		EditorTrailingGraph:    "E488: Trailing characters: %s (use :graph or :graph all)",
		EditorTrailingSync:     "E488: Trailing characters: %s (use :sync, :sync status or :sync setup)",
		EditorBadStatus:        "E474: Invalid argument: %s (use active, onhold, completed or dropped)",
		EditorNotInTable:       "Not in a table",
		EditorClipboardEmpty:   "Clipboard is empty or unavailable",
		EditorPasteNeeds:       "Paste image needs %s",
		EditorNoNote:           "No note open",
		EditorHintNewNote:      "new note",
		EditorHintSearch:       "search",
		EditorHintExplorer:     "explorer",
		EditorPlaceholder:      "Start writing… (i to insert)",
		EditorNoMatches:        "No matches",
		EditorNoResults:        "No results",
		EditorNoPreviousSearch: "E35: No previous regular expression",
		EditorPatternNotFound:  "E486: Pattern not found: %s",
		EditorSearchWrappedTop: "search hit BOTTOM, continuing at TOP",
		EditorSearchWrappedBot: "search hit TOP, continuing at BOTTOM",
		EditorUndoTooLarge:     "Note too large to keep undo history (limit %d MB)",
		EditorUndoOldest:       "Already at oldest change",
		EditorUndoNewest:       "Already at newest change",
		EditorUndoBefore:       "1 change; before #%d",
		EditorUndoAfter:        "1 change; after #%d",
		EditorNewColumn:        "Column",

		// Starter text the "/" menu inserts into the note (A6). Fence
		// names, keys such as "title:" or "(choice)" and flowchart node
		// IDs are syntax and stay as they are.
		SlashColumnN:       "Column %d",
		SlashBoardStart:    "```board\n## Not started\n- ",
		SlashBoardRest:     "\n## In progress\n- Card 2\n  Lines under a card are its description\n## Done\n- Card 3\n```",
		SlashTimelineRest:  "Card 1: %s -> %s\nCard 2: %s -> %s\n```",
		SlashChartRest:     "\nJan: 12\nFeb: 20\nMar: 7\nApr: 15\n```",
		SlashFormRest:      "\ndescription: Optional description\n? Question 1\n? Question 2 (choice): Option 1 | Option 2 | Option 3\n```",
		SlashFlowStart:     "```flow\nstart([Start]) --> ",
		SlashFlowRest:      "check{Is it working?}\ncheck -->|yes| done([Ship it])\ncheck -->|no| fix[Fix it]\nfix --> check\n```",
		SlashHintClipboard: "clipboard",

		DashNewNote:       "New Note",
		DashFindNote:      "Find Note",
		DashOpenExplorer:  "Open Explorer",
		DashThemes:        "Themes",
		DashKeymaps:       "Keymaps",
		DashGraph:         "Graph",
		DashSettings:      "Settings",
		DashQuit:          "Quit",
		DashSync:          "Sync between computers",
		DashTagline:       "notes that live in plain Markdown files",
		DashRecent:        "Recent",
		DashFolders:       "Folders",
		DashNotes + one:   "%d note",
		DashNotes + other: "%d notes",
		DashFooterHint:    "SPC opens every other command · ? shows them all",

		PreviewTop:                      "Top",
		PreviewBottom:                   "Bot",
		PreviewNothing:                  "Nothing to preview",
		PreviewRenderError:              "Error rendering preview: %v",
		PreviewHiddenLines + one:        "… %d lines", // today's English, kept as it was
		PreviewHiddenLines + other:      "… %d lines",
		PreviewCodeLabel:                "code",
		PreviewCodeInfo + one:           "%s · %d line",
		PreviewCodeInfo + other:         "%s · %d lines",
		PreviewAddRow:                   "+ Add row",
		PreviewAddColumn:                "+ Add column",
		PreviewImage:                    "image",
		PreviewNotFound:                 "(not found)",
		PreviewNoSuchNote:               "(no such note)",
		PreviewSectionNotFound:          "section not found",
		PreviewMentionButton:            "[ link ]",
		PreviewUnlinkedMentions + one:   " Unlinked mentions in %d note ",
		PreviewUnlinkedMentions + other: " Unlinked mentions in %d notes ",
		PreviewMentions + one:           "%d mention",
		PreviewMentions + other:         "%d mentions",
		PreviewLinkedFrom + one:         " Linked from %d note ",
		PreviewLinkedFrom + other:       " Linked from %d notes ",
		PreviewBoardCards:               "Cards",
		PreviewBoardEmpty:               `(empty board: add "## Column" headings and "- card" lines)`,
		PreviewNewPage:                  "+ New page",
		PreviewCalendarEvents + one:     "%d events", // today's English, kept as it was
		PreviewCalendarEvents + other:   "%d events",
		PreviewCalendarMore:             "+%d more",
		PreviewToday:                    "Today",
		PreviewWeekdays:                 "Sun,Mon,Tue,Wed,Thu,Fri,Sat",
		PreviewMonths:                   "January,February,March,April,May,June,July,August,September,October,November,December",
		PreviewMonthsShort:              "Jan,Feb,Mar,Apr,May,Jun,Jul,Aug,Sep,Oct,Nov,Dec",
		PreviewChartEmpty:               `(empty chart: add "Label: 12" lines)`,
		PreviewAddValue:                 "+ Add value",
		PreviewTimelineEmpty:            `(empty timeline: add "Task: 2026-10-01 -> 2026-10-05" lines)`,
		PreviewTimelineNew:              "+ New",
		PreviewFormTitle:                "Form title",
		PreviewFormDescription:          "Description (optional)",
		PreviewFormUpTo:                 "(Respondents can select up to %d)",
		PreviewFormAddOption:            "+ Add option",
		PreviewFormDate:                 "Pick a date (YYYY-MM-DD)",
		PreviewFormLong:                 "Long answer",
		PreviewFormEmail:                "name@example.com",
		PreviewFormAnswer:               "Respondent's answer",
		PreviewFormNoQuestions:          `(add questions: "? Question", "? Pick one (choice): A | B")`,
		PreviewFormAddQuestion:          "⊕ add question",
		PreviewFlowEmpty:                "Empty flowchart: write lines like  a[Start] --> b[Next]",
		PreviewMathEmpty:                "Empty equation: write LaTeX like  E = mc^2",
		PreviewImageMissing:             "not found",
		PreviewImageUnsupported:         "unsupported image",
		PreviewImageEmpty:               "empty image",
		PreviewImageRemote:              "remote image",
		PreviewImageNoFolder:            "unknown folder",

		SidebarFindNote:        "Find note",
		SidebarEmpty:           "Empty workspace — press n",
		SidebarEditTitle:       "EDIT",
		SidebarEditHint:        "= apply · esc leave",
		SidebarEditEmpty:       "Empty — o adds a line",
		SidebarNamePlaceholder: "name",
	})
	register(ES, map[Key]string{
		KeymapMoreGroup:        "más",
		KeymapGroup("note"):    "nota",
		KeymapGroup("view"):    "vista",
		KeymapGroup("go"):      "ir",
		KeysUnknownAction:      "keys: acción desconocida %q",
		KeysSeveralNeedSpace:   "%q: un atajo de varias teclas debe empezar con space",
		KeysOneKey:             "%q: usa una tecla, o space y una o dos teclas",
		KeysBoundTwice:         "keys: %q está asignada a %s y a %s; quita una de las dos",
		KeysPrefixClash:        "keys: %q de %s es el inicio de %q de %s, así que %s nunca se puede teclear; quita una de las dos",
		KeysLeaderInExplorer:   "keys: %q de %s empieza una secuencia de la tecla líder, así que el explorador nunca la recibe; elige otra tecla",
		KeysShadowedOnExplorer: "keys: %q está asignada a %s y a %s, y %s se ejecuta primero incluso en el explorador; quita una de las dos",

		EditorNotACommand:      "E492: No es una orden del editor: %s",
		EditorTrailing:         "E488: Sobran caracteres: %s (usa %s)",
		EditorTrailingGraph:    "E488: Sobran caracteres: %s (usa :graph o :graph all)",
		EditorTrailingSync:     "E488: Sobran caracteres: %s (usa :sync, :sync status o :sync setup)",
		EditorBadStatus:        "E474: Argumento no válido: %s (usa active, onhold, completed o dropped)",
		EditorNotInTable:       "No estás en una tabla",
		EditorClipboardEmpty:   "El portapapeles está vacío o no está disponible",
		EditorPasteNeeds:       "Pegar una imagen necesita %s",
		EditorNoNote:           "Ninguna nota abierta",
		EditorHintNewNote:      "nueva nota",
		EditorHintSearch:       "buscar",
		EditorHintExplorer:     "explorador",
		EditorPlaceholder:      "Empieza a escribir… (i para insertar)",
		EditorNoMatches:        "Sin coincidencias",
		EditorNoResults:        "Sin resultados",
		EditorNoPreviousSearch: "E35: No hay una expresión regular anterior",
		EditorPatternNotFound:  "E486: Patrón no encontrado: %s",
		EditorSearchWrappedTop: "la búsqueda llegó al FINAL, sigue desde el PRINCIPIO",
		EditorSearchWrappedBot: "la búsqueda llegó al PRINCIPIO, sigue desde el FINAL",
		EditorUndoTooLarge:     "La nota es demasiado grande para guardar el historial de deshacer (límite %d MB)",
		EditorUndoOldest:       "Ya estás en el cambio más antiguo",
		EditorUndoNewest:       "Ya estás en el cambio más reciente",
		EditorUndoBefore:       "1 cambio; antes de #%d",
		EditorUndoAfter:        "1 cambio; después de #%d",
		EditorNewColumn:        "Columna",

		SlashColumnN:       "Columna %d",
		SlashBoardStart:    "```board\n## Sin empezar\n- ",
		SlashBoardRest:     "\n## En curso\n- Tarjeta 2\n  Las líneas bajo una tarjeta son su descripción\n## Hecho\n- Tarjeta 3\n```",
		SlashTimelineRest:  "Tarjeta 1: %s -> %s\nTarjeta 2: %s -> %s\n```",
		SlashChartRest:     "\nEne: 12\nFeb: 20\nMar: 7\nAbr: 15\n```",
		SlashFormRest:      "\ndescription: Descripción opcional\n? Pregunta 1\n? Pregunta 2 (choice): Opción 1 | Opción 2 | Opción 3\n```",
		SlashFlowStart:     "```flow\nstart([Inicio]) --> ",
		SlashFlowRest:      "check{¿Funciona?}\ncheck -->|sí| done([Publicarlo])\ncheck -->|no| fix[Arreglarlo]\nfix --> check\n```",
		SlashHintClipboard: "portapapeles",

		DashNewNote:       "Nueva nota",
		DashFindNote:      "Buscar nota",
		DashOpenExplorer:  "Abrir explorador",
		DashThemes:        "Temas",
		DashKeymaps:       "Atajos",
		DashGraph:         "Grafo",
		DashSettings:      "Ajustes",
		DashQuit:          "Salir",
		DashSync:          "Sincronizar entre computadoras",
		DashTagline:       "notas que viven en archivos Markdown simples",
		DashRecent:        "Recientes",
		DashFolders:       "Carpetas",
		DashNotes + one:   "%d nota",
		DashNotes + other: "%d notas",
		DashFooterHint:    "SPC abre los demás comandos · ? los muestra todos",

		PreviewTop:                      "Arriba",
		PreviewBottom:                   "Abajo",
		PreviewNothing:                  "Nada que mostrar",
		PreviewRenderError:              "Error al dibujar el preview: %v",
		PreviewHiddenLines + one:        "… %d línea",
		PreviewHiddenLines + other:      "… %d líneas",
		PreviewCodeLabel:                "código",
		PreviewCodeInfo + one:           "%s · %d línea",
		PreviewCodeInfo + other:         "%s · %d líneas",
		PreviewAddRow:                   "+ Añadir fila",
		PreviewAddColumn:                "+ Añadir columna",
		PreviewImage:                    "imagen",
		PreviewNotFound:                 "(no encontrada)",
		PreviewNoSuchNote:               "(no existe esa nota)",
		PreviewSectionNotFound:          "sección no encontrada",
		PreviewMentionButton:            "[ enlazar ]",
		PreviewUnlinkedMentions + one:   " Menciones sin enlazar en %d nota ",
		PreviewUnlinkedMentions + other: " Menciones sin enlazar en %d notas ",
		PreviewMentions + one:           "%d mención",
		PreviewMentions + other:         "%d menciones",
		PreviewLinkedFrom + one:         " Enlazada desde %d nota ",
		PreviewLinkedFrom + other:       " Enlazada desde %d notas ",
		PreviewBoardCards:               "Tarjetas",
		PreviewBoardEmpty:               `(tablero vacío: añade títulos "## Columna" y líneas "- tarjeta")`,
		PreviewNewPage:                  "+ Nueva página",
		PreviewCalendarEvents + one:     "%d evento",
		PreviewCalendarEvents + other:   "%d eventos",
		PreviewCalendarMore:             "+%d más",
		PreviewToday:                    "Hoy",
		PreviewWeekdays:                 "Dom,Lun,Mar,Mié,Jue,Vie,Sáb",
		PreviewMonths:                   "Enero,Febrero,Marzo,Abril,Mayo,Junio,Julio,Agosto,Septiembre,Octubre,Noviembre,Diciembre",
		PreviewMonthsShort:              "Ene,Feb,Mar,Abr,May,Jun,Jul,Ago,Sep,Oct,Nov,Dic",
		PreviewChartEmpty:               `(gráfico vacío: añade líneas "Etiqueta: 12")`,
		PreviewAddValue:                 "+ Añadir valor",
		PreviewTimelineEmpty:            `(cronograma vacío: añade líneas "Tarea: 2026-10-01 -> 2026-10-05")`,
		PreviewTimelineNew:              "+ Nueva",
		PreviewFormTitle:                "Título del formulario",
		PreviewFormDescription:          "Descripción (opcional)",
		PreviewFormUpTo:                 "(Se pueden elegir hasta %d)",
		PreviewFormAddOption:            "+ Añadir opción",
		PreviewFormDate:                 "Elige una fecha (AAAA-MM-DD)",
		PreviewFormLong:                 "Respuesta larga",
		PreviewFormEmail:                "nombre@ejemplo.com",
		PreviewFormAnswer:               "Respuesta",
		PreviewFormNoQuestions:          `(añade preguntas: "? Pregunta", "? Elige una (choice): A | B")`,
		PreviewFormAddQuestion:          "⊕ añadir pregunta",
		PreviewFlowEmpty:                "Diagrama de flujo vacío: escribe líneas como  a[Inicio] --> b[Siguiente]",
		PreviewMathEmpty:                "Ecuación vacía: escribe LaTeX como  E = mc^2",
		PreviewImageMissing:             "no encontrada",
		PreviewImageUnsupported:         "imagen no compatible",
		PreviewImageEmpty:               "imagen vacía",
		PreviewImageRemote:              "imagen remota",
		PreviewImageNoFolder:            "carpeta desconocida",

		SidebarFindNote:        "Buscar nota",
		SidebarEmpty:           "Vault vacío — pulsa n",
		SidebarEditTitle:       "EDITAR",
		SidebarEditHint:        "= aplicar · esc salir",
		SidebarEditEmpty:       "Vacía — o añade una línea",
		SidebarNamePlaceholder: "nombre",
	})
}
