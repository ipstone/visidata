package sheet

import "strconv"

func (s *Sheet) ColumnsSheet() *Sheet {
	sh := New("columns:"+s.Name, s.Source, []string{"index", "name", "type", "width", "hidden", "visible"})
	sh.MetaKind = "columns"
	sh.MetaTargets = []*Sheet{s}
	sh.Columns[0].Kind = KindInt
	sh.Columns[3].Kind = KindInt
	sh.Columns[4].Kind = KindBool
	sh.Columns[5].Kind = KindBool

	for i, col := range s.Columns {
		visible := !col.Hidden
		width := col.Width
		if width <= 0 {
			width = len(col.Label())
		}
		sh.MetaRows = append(sh.MetaRows, i)
		sh.AddRawRow([]string{
			strconv.Itoa(i + 1),
			col.Name,
			string(col.EffectiveKind()),
			strconv.Itoa(width),
			strconv.FormatBool(col.Hidden),
			strconv.FormatBool(visible),
		})
	}

	return sh
}

func (s *Sheet) OptionsSheet() *Sheet {
	sh := New("options:"+s.Name, s.Source, []string{"option", "value"})

	sortColumn := ""
	if s.SortState.Direction != SortNone && s.SortState.ColumnIndex >= 0 && s.SortState.ColumnIndex < len(s.Columns) {
		sortColumn = s.Columns[s.SortState.ColumnIndex].Name
	}

	for _, row := range []Row{
		{"name", s.Name},
		{"source", s.Source},
		{"rows", strconv.Itoa(len(s.Rows))},
		{"columns", strconv.Itoa(len(s.Columns))},
		{"visible_columns", strconv.Itoa(len(s.VisibleColumnIndices()))},
		{"hidden_columns", strconv.Itoa(s.HiddenColumnCount())},
		{"selected_rows", strconv.Itoa(s.SelectedCount())},
		{"cursor_row", strconv.Itoa(s.CursorRow + 1)},
		{"cursor_col", strconv.Itoa(visibleColumnOrdinalValue(s, s.CursorCol))},
		{"sort_column", sortColumn},
		{"sort_direction", string(s.SortState.Direction)},
		{"search_query", s.SearchState.Query},
		{"search_matches", strconv.Itoa(len(s.SearchState.Matches))},
		{"save_path", s.SuggestedSavePath()},
	} {
		sh.AddRawRow(row)
	}

	return sh
}

func visibleColumnOrdinalValue(s *Sheet, col int) int {
	ordinal := 0
	for i, column := range s.Columns {
		if column.Hidden {
			continue
		}
		ordinal++
		if i == col {
			return ordinal
		}
	}
	return 0
}
