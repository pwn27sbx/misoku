package i18n

// The menu of a table cell in the preview.
const (
	TableEditCell     Key = "table.edit_cell"
	TableRenameColumn Key = "table.rename_column"
	TableAddRowBelow  Key = "table.add_row_below"
	TableAddColRight  Key = "table.add_column_right"
	TableDeleteColumn Key = "table.delete_column"
	TableDeleteRow    Key = "table.delete_row"
	TableEmptyCell    Key = "table.empty_cell"
)

func init() {
	register(EN, map[Key]string{
		TableEditCell:     "Edit cell",
		TableRenameColumn: "Rename column",
		TableAddRowBelow:  "Add row below",
		TableAddColRight:  "Add column right",
		TableDeleteColumn: "Delete column",
		TableDeleteRow:    "Delete row",
		TableEmptyCell:    "(empty cell)",
	})
	register(ES, map[Key]string{
		TableEditCell:     "Editar celda",
		TableRenameColumn: "Renombrar columna",
		TableAddRowBelow:  "Añadir fila debajo",
		TableAddColRight:  "Añadir columna a la derecha",
		TableDeleteColumn: "Eliminar columna",
		TableDeleteRow:    "Eliminar fila",
		TableEmptyCell:    "(celda vacía)",
	})
}
