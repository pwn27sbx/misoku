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
	{File: "B00 Básico.md", Title: "Básico"},
	{File: "B01 Títulos.md", Title: "Títulos", Check: "subheading"},
	{File: "B02 Casillas.md", Title: "Casillas", Check: "checkbox"},
	{File: "B03 Menú de bloques.md", Title: "Menú de bloques", Check: "table"},
	{File: "B04 Links.md", Title: "Links", Check: "wikilink"},
	{File: "B05 Links a títulos.md", Title: "Links a títulos", Check: "headinglink"},
	{File: "B06 Referencias a bloque.md", Title: "Referencias a bloque", Check: "blockref"},
	{File: "B07 Explorador.md", Title: "Explorador", Check: "newnote"},
}
