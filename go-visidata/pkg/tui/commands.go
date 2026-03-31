package tui

type commandInfo struct {
	Keys string
	Name string
	Help string
}

var commandCatalog = []commandInfo{
	{Keys: "q", Name: "quit", Help: "quit or close the current derived sheet"},
	{Keys: "Enter", Name: "open-row", Help: "open the current row as a column/value table"},
	{Keys: "/", Name: "search", Help: "search for a case-insensitive substring"},
	{Keys: "[ , ]", Name: "sort", Help: "sort the current column ascending or descending"},
	{Keys: "=", Name: "expr-col", Help: "add a materialized expression column"},
	{Keys: "&", Name: "join", Help: "join with the previous sheet in the stack using the current column"},
	{Keys: "f", Name: "freeze", Help: "open a frozen snapshot of the current sheet"},
	{Keys: "m", Name: "melt", Help: "open an unpivoted view using the current column as the identifier"},
	{Keys: "D", Name: "dedupe", Help: "open a deduplicated view using the current column"},
	{Keys: "F", Name: "frequency", Help: "open a frequency sheet for the current column"},
	{Keys: "I", Name: "describe", Help: "open a describe sheet for the current sheet"},
	{Keys: "W", Name: "pivot", Help: "open a pivot view grouped by the current column"},
	{Keys: "T", Name: "transpose", Help: "transpose the current sheet"},
	{Keys: "V", Name: "sheets", Help: "open the current sheet stack"},
	{Keys: "M", Name: "columns", Help: "open an editable columns metasheet"},
	{Keys: "O", Name: "options", Help: "open a viewer state sheet"},
	{Keys: "P", Name: "cmdlog", Help: "open the in-memory command log for this session"},
	{Keys: "?", Name: "commands", Help: "open the command reference"},
	{Keys: "R", Name: "redo", Help: "redo the last undone sheet mutation for the current sheet"},
	{Keys: "U", Name: "undo", Help: "undo the last sheet mutation for the current sheet"},
	{Keys: "s / t / u", Name: "select", Help: "toggle, select all, or clear row selection"},
	{Keys: "c / C", Name: "copy", Help: "copy the current cell or selected rows"},
	{Keys: "e", Name: "edit-cell", Help: "edit the current cell in place"},
	{Keys: "d", Name: "delete", Help: "delete selected rows or the current row"},
	{Keys: "S", Name: "save", Help: "save to a suggested export path"},
	{Keys: "- / H", Name: "columns-visibility", Help: "hide the current column or reveal all columns"},
	{Keys: "^ / _", Name: "columns-edit", Help: "rename the current column or set its width"},
	{Keys: "~ # % $ @", Name: "type-override", Help: "override the current column display type"},
	{Keys: "| / \\", Name: "regex-select", Help: "select or unselect rows matching a regex in the current column"},
}

func controlHints() []string {
	hints := make([]string, 0, len(commandCatalog))
	for _, command := range commandCatalog {
		hints = append(hints, command.Keys+" "+command.Name)
	}
	return hints
}
