package tui

type commandInfo struct {
	Keys string
	Name string
	Help string
}

var commandCatalog = []commandInfo{
	{Keys: "q", Name: "quit", Help: "quit or close the current derived sheet"},
	{Keys: "/", Name: "search", Help: "search for a case-insensitive substring"},
	{Keys: "[ , ]", Name: "sort", Help: "sort the current column ascending or descending"},
	{Keys: "f", Name: "freeze", Help: "open a frozen snapshot of the current sheet"},
	{Keys: "D", Name: "dedupe", Help: "open a deduplicated view using the current column"},
	{Keys: "F", Name: "frequency", Help: "open a frequency sheet for the current column"},
	{Keys: "I", Name: "describe", Help: "open a describe sheet for the current sheet"},
	{Keys: "T", Name: "transpose", Help: "transpose the current sheet"},
	{Keys: "V", Name: "sheets", Help: "open the current sheet stack"},
	{Keys: "M", Name: "columns", Help: "open a columns metasheet"},
	{Keys: "O", Name: "options", Help: "open a viewer state sheet"},
	{Keys: "?", Name: "commands", Help: "open the command reference"},
	{Keys: "s / t / u", Name: "select", Help: "toggle, select all, or clear row selection"},
	{Keys: "c / C", Name: "copy", Help: "copy the current cell or selected rows"},
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
