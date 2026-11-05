package tutor

// Lesson is one note of the tutorial vault.
type Lesson struct {
	// File is the note's file name inside the tutorial vault.
	File string
	// Title is the level's name as shown to the player.
	Title string
	// Check names the rule Passed applies to the note; "" means no check.
	Check string
}

// Lessons lists the tutorial notes in play order. The index (position 0) has
// no check; every other position is its level number.
var Lessons = []Lesson{
	{File: "00 Empieza aquí.md", Title: "Empieza aquí"},
	{File: "01 Títulos.md", Title: "Títulos", Check: "subheading"},
	{File: "02 Casillas.md", Title: "Casillas", Check: "checkbox"},
	{File: "03 Menú de bloques.md", Title: "Menú de bloques", Check: "table"},
	{File: "04 Links.md", Title: "Links", Check: "wikilink"},
	{File: "05 Links a títulos.md", Title: "Links a títulos", Check: "headinglink"},
	{File: "06 Referencias a bloque.md", Title: "Referencias a bloque", Check: "blockref"},
	{File: "07 Explorador.md", Title: "Explorador", Check: "newnote"},
}
