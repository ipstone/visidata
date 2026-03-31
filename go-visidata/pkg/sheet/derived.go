package sheet

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func (s *Sheet) FreezeSheet() *Sheet {
	visibleCols := s.VisibleColumnIndices()
	headers := make([]string, 0, len(visibleCols))
	for _, columnIndex := range visibleCols {
		headers = append(headers, s.Columns[columnIndex].Name)
	}

	sh := New(fmt.Sprintf("freeze:%s", s.Name), s.Source, headers)
	for i, columnIndex := range visibleCols {
		sh.Columns[i].Kind = s.Columns[columnIndex].EffectiveKind()
		sh.Columns[i].Width = s.Columns[columnIndex].Width
	}
	for _, row := range s.Rows {
		sh.AddRawRow(visibleRowValues(row, visibleCols))
	}

	return sh
}

func (s *Sheet) FilterValueSheet(columnIndex int, value string) (*Sheet, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}

	headers := make([]string, len(s.Columns))
	for i, col := range s.Columns {
		headers[i] = col.Name
	}

	sh := New(fmt.Sprintf("filter:%s:%s=%s", s.Name, s.Columns[columnIndex].Name, value), s.Source, headers)
	sh.MetaKind = "filter"
	sh.MetaCols = []int{columnIndex}
	sh.MetaTargets = []*Sheet{s}
	for i, col := range s.Columns {
		sh.Columns[i] = col
	}

	for rowIndex, row := range s.Rows {
		if s.Cell(rowIndex, columnIndex) != value {
			continue
		}
		sh.AddRawRow(row)
	}

	return sh, nil
}

func (s *Sheet) DedupeSheet(columnIndex int) (*Sheet, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}

	visibleCols := s.VisibleColumnIndices()
	headers := make([]string, 0, len(visibleCols)+1)
	for _, visibleIndex := range visibleCols {
		headers = append(headers, s.Columns[visibleIndex].Name)
	}
	headers = append(headers, "duplicate_count")

	type dedupeEntry struct {
		rowIndex int
		count    int
	}

	entries := make(map[string]*dedupeEntry)
	order := make([]string, 0)
	for rowIndex := range s.Rows {
		key := s.Cell(rowIndex, columnIndex)
		if entry, ok := entries[key]; ok {
			entry.count++
			continue
		}
		entries[key] = &dedupeEntry{rowIndex: rowIndex, count: 1}
		order = append(order, key)
	}

	sh := New(fmt.Sprintf("dedupe:%s:%s", s.Name, s.Columns[columnIndex].Name), s.Source, headers)
	for i, visibleIndex := range visibleCols {
		sh.Columns[i].Kind = s.Columns[visibleIndex].EffectiveKind()
		sh.Columns[i].Width = s.Columns[visibleIndex].Width
	}
	sh.Columns[len(sh.Columns)-1].Kind = KindInt

	for _, key := range order {
		entry := entries[key]
		row := append(visibleRowValues(s.Rows[entry.rowIndex], visibleCols), strconv.Itoa(entry.count))
		sh.AddRawRow(row)
	}

	return sh, nil
}

func (s *Sheet) PivotSheet(columnIndex int) (*Sheet, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}

	type aggColumn struct {
		index int
		name  string
		kind  ValueKind
	}
	type pivotRow struct {
		key   string
		count int
		sums  []float64
	}

	aggCols := make([]aggColumn, 0)
	for _, visibleIndex := range s.VisibleColumnIndices() {
		if visibleIndex == columnIndex {
			continue
		}
		kind := s.Columns[visibleIndex].EffectiveKind()
		if kind == KindInt || kind == KindFloat || kind == KindCurrency {
			aggCols = append(aggCols, aggColumn{
				index: visibleIndex,
				name:  s.Columns[visibleIndex].Name + "_sum",
				kind:  kind,
			})
		}
	}

	headers := []string{s.Columns[columnIndex].Name, "count"}
	for _, aggCol := range aggCols {
		headers = append(headers, aggCol.name)
	}

	groups := make(map[string]*pivotRow)
	order := make([]string, 0)
	for rowIndex := range s.Rows {
		key := s.Cell(rowIndex, columnIndex)
		group, ok := groups[key]
		if !ok {
			group = &pivotRow{key: key, sums: make([]float64, len(aggCols))}
			groups[key] = group
			order = append(order, key)
		}
		group.count++
		for i, aggCol := range aggCols {
			if value, ok := parseNumericValue(aggCol.kind, s.Cell(rowIndex, aggCol.index)); ok {
				group.sums[i] += value
			}
		}
	}

	kind := s.Columns[columnIndex].EffectiveKind()
	sort.Slice(order, func(i, j int) bool {
		return compareValues(kind, order[i], order[j]) < 0
	})

	sh := New(fmt.Sprintf("pivot:%s:%s", s.Name, s.Columns[columnIndex].Name), s.Source, headers)
	sh.Columns[0].Kind = kind
	sh.Columns[1].Kind = KindInt
	for i, aggCol := range aggCols {
		sh.Columns[i+2].Kind = aggCol.kind
	}

	for _, key := range order {
		group := groups[key]
		row := []string{group.key, strconv.Itoa(group.count)}
		for i, aggCol := range aggCols {
			row = append(row, formatNumericValue(aggCol.kind, group.sums[i]))
		}
		sh.AddRawRow(row)
	}

	return sh, nil
}

func (s *Sheet) MeltSheet(idColumnIndex int) (*Sheet, error) {
	if idColumnIndex < 0 || idColumnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", idColumnIndex)
	}

	visibleCols := s.VisibleColumnIndices()
	valueCols := make([]int, 0, len(visibleCols)-1)
	for _, colIndex := range visibleCols {
		if colIndex == idColumnIndex {
			continue
		}
		valueCols = append(valueCols, colIndex)
	}

	sh := New(fmt.Sprintf("melt:%s:%s", s.Name, s.Columns[idColumnIndex].Name), s.Source, []string{s.Columns[idColumnIndex].Name, "variable", "value"})
	sh.Columns[0].Kind = s.Columns[idColumnIndex].EffectiveKind()
	sh.Columns[1].Kind = KindString
	sh.Columns[2].Kind = KindString

	for rowIndex := range s.Rows {
		idValue := s.Cell(rowIndex, idColumnIndex)
		for _, colIndex := range valueCols {
			sh.AddRawRow([]string{idValue, s.Columns[colIndex].Name, s.Cell(rowIndex, colIndex)})
		}
	}

	return sh, nil
}

func (s *Sheet) JoinSheet(other *Sheet, leftColumnIndex, rightColumnIndex int) (*Sheet, error) {
	if other == nil {
		return nil, fmt.Errorf("join requires another sheet")
	}
	if leftColumnIndex < 0 || leftColumnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", leftColumnIndex)
	}
	if rightColumnIndex < 0 || rightColumnIndex >= len(other.Columns) {
		return nil, fmt.Errorf("column %d out of range", rightColumnIndex)
	}

	leftVisible := s.VisibleColumnIndices()
	rightVisible := other.VisibleColumnIndices()
	rightNames := make(map[string]struct{}, len(rightVisible))
	for _, columnIndex := range rightVisible {
		rightNames[other.Columns[columnIndex].Name] = struct{}{}
	}

	headers := make([]string, 0, len(leftVisible)+len(rightVisible))
	leftKinds := make([]ValueKind, 0, len(leftVisible))
	rightKinds := make([]ValueKind, 0, len(rightVisible))
	for _, columnIndex := range leftVisible {
		name := s.Columns[columnIndex].Name
		if _, exists := rightNames[name]; exists {
			name = "left." + name
		}
		headers = append(headers, name)
		leftKinds = append(leftKinds, s.Columns[columnIndex].EffectiveKind())
	}
	leftNames := make(map[string]struct{}, len(leftVisible))
	for _, columnIndex := range leftVisible {
		leftNames[s.Columns[columnIndex].Name] = struct{}{}
	}
	for _, columnIndex := range rightVisible {
		name := other.Columns[columnIndex].Name
		if _, exists := leftNames[name]; exists {
			name = "right." + name
		}
		headers = append(headers, name)
		rightKinds = append(rightKinds, other.Columns[columnIndex].EffectiveKind())
	}

	index := make(map[string][]Row)
	for _, row := range other.Rows {
		key := cellAt(row, rightColumnIndex)
		index[key] = append(index[key], row)
	}

	joinName := fmt.Sprintf("join:%s+%s:%s=%s", s.Name, other.Name, s.Columns[leftColumnIndex].Name, other.Columns[rightColumnIndex].Name)
	sh := New(joinName, s.Source, headers)
	for i, kind := range leftKinds {
		sh.Columns[i].Kind = kind
	}
	for i, kind := range rightKinds {
		sh.Columns[len(leftKinds)+i].Kind = kind
	}

	blankRight := make([]string, len(rightVisible))
	for _, leftRow := range s.Rows {
		leftValues := visibleRowValues(leftRow, leftVisible)
		matches := index[cellAt(leftRow, leftColumnIndex)]
		if len(matches) == 0 {
			sh.AddRawRow(append(append([]string{}, leftValues...), blankRight...))
			continue
		}
		for _, rightRow := range matches {
			joined := append(append([]string{}, leftValues...), visibleRowValues(rightRow, rightVisible)...)
			sh.AddRawRow(joined)
		}
	}

	return sh, nil
}

func (s *Sheet) FrequencySheet(columnIndex int) (*Sheet, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}

	type entry struct {
		value string
		count int
	}

	counts := make(map[string]int)
	for rowIndex := range s.Rows {
		counts[s.Cell(rowIndex, columnIndex)]++
	}

	entries := make([]entry, 0, len(counts))
	for value, count := range counts {
		entries = append(entries, entry{value: value, count: count})
	}

	kind := s.Columns[columnIndex].EffectiveKind()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return compareValues(kind, entries[i].value, entries[j].value) < 0
	})

	sh := New(fmt.Sprintf("freq:%s:%s", s.Name, s.Columns[columnIndex].Name), s.Source, []string{s.Columns[columnIndex].Name, "count", "percent"})
	sh.MetaKind = "freq"
	sh.MetaCols = []int{columnIndex}
	sh.MetaTargets = []*Sheet{s}
	sh.Columns[0].Kind = kind
	sh.Columns[1].Kind = KindInt
	sh.Columns[2].Kind = KindFloat

	total := len(s.Rows)
	for _, item := range entries {
		percent := 0.0
		if total > 0 {
			percent = (float64(item.count) / float64(total)) * 100
		}
		sh.AddRawRow([]string{
			item.value,
			strconv.Itoa(item.count),
			strconv.FormatFloat(percent, 'f', 2, 64),
		})
	}

	return sh, nil
}

func (s *Sheet) DescribeSheet() *Sheet {
	sh := New(fmt.Sprintf("describe:%s", s.Name), s.Source, []string{"column", "type", "rows", "nulls", "distinct", "min", "max", "sum", "avg"})
	sh.Columns[2].Kind = KindInt
	sh.Columns[3].Kind = KindInt
	sh.Columns[4].Kind = KindInt
	sh.Columns[7].Kind = KindFloat
	sh.Columns[8].Kind = KindFloat

	for _, columnIndex := range s.VisibleColumnIndices() {
		col := s.Columns[columnIndex]
		kind := col.EffectiveKind()
		nulls := 0
		distinct := make(map[string]struct{})
		minValue := ""
		maxValue := ""
		hasValue := false
		numericTotal := 0.0
		numericCount := 0

		for rowIndex := range s.Rows {
			value := strings.TrimSpace(s.Cell(rowIndex, columnIndex))
			if value == "" {
				nulls++
				continue
			}

			distinct[value] = struct{}{}
			if !hasValue || compareValues(kind, value, minValue) < 0 {
				minValue = value
			}
			if !hasValue || compareValues(kind, value, maxValue) > 0 {
				maxValue = value
			}
			hasValue = true

			if numericValue, ok := parseNumericValue(kind, value); ok {
				numericTotal += numericValue
				numericCount++
			}
		}

		sumValue := ""
		avgValue := ""
		if numericCount > 0 {
			sumValue = formatNumericValue(kind, numericTotal)
			avgValue = strconv.FormatFloat(numericTotal/float64(numericCount), 'f', 2, 64)
		}

		sh.AddRawRow([]string{
			col.Name,
			string(kind),
			strconv.Itoa(len(s.Rows)),
			strconv.Itoa(nulls),
			strconv.Itoa(len(distinct)),
			minValue,
			maxValue,
			sumValue,
			avgValue,
		})
	}

	return sh
}

func (s *Sheet) TransposeSheet() *Sheet {
	visibleCols := s.VisibleColumnIndices()
	headers := make([]string, 0, len(s.Rows)+1)
	headers = append(headers, "column")
	for rowIndex := range s.Rows {
		headers = append(headers, fmt.Sprintf("row_%d", rowIndex+1))
	}

	sh := New(fmt.Sprintf("transpose:%s", s.Name), s.Source, headers)
	for _, columnIndex := range visibleCols {
		row := make([]string, 0, len(s.Rows)+1)
		row = append(row, s.Columns[columnIndex].Name)
		for rowIndex := range s.Rows {
			row = append(row, s.Cell(rowIndex, columnIndex))
		}
		sh.AddRawRow(row)
	}

	return sh
}

func parseNumericValue(kind ValueKind, value string) (float64, bool) {
	switch kind {
	case KindInt:
		parsed, err := strconv.ParseInt(normalizeNumber(value), 10, 64)
		if err != nil {
			return 0, false
		}
		return float64(parsed), true
	case KindFloat, KindCurrency:
		parsed, err := strconv.ParseFloat(normalizeNumber(value), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func formatNumericValue(kind ValueKind, value float64) string {
	if kind == KindInt {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}
