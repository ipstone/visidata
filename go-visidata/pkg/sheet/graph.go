package sheet

import (
	"fmt"
	"strconv"
)

func (s *Sheet) GraphSheet(columnIndex int) (*Sheet, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}
	if len(s.Rows) == 0 {
		return nil, fmt.Errorf("graph needs at least one row")
	}

	current := s.Columns[columnIndex]
	kind := current.EffectiveKind()
	if isNumericKind(kind) {
		if otherIndex := s.otherNumericColumn(columnIndex); otherIndex >= 0 {
			return s.scatterGraphSheet(columnIndex, otherIndex)
		}
		return s.lineGraphSheet(columnIndex)
	}

	valueIndex := s.firstNumericColumnExcluding(columnIndex)
	if valueIndex < 0 {
		return nil, fmt.Errorf("graph needs a numeric column")
	}
	return s.barGraphSheet(columnIndex, valueIndex)
}

func (s *Sheet) barGraphSheet(labelIndex, valueIndex int) (*Sheet, error) {
	valueKind := s.Columns[valueIndex].EffectiveKind()
	headers := []string{s.Columns[labelIndex].Name, s.Columns[valueIndex].Name, "x"}
	sh := New(fmt.Sprintf("graph:%s:%s", s.Name, s.Columns[valueIndex].Name), s.Source, headers)
	sh.MetaKind = "graph"
	sh.MetaCols = []int{labelIndex, valueIndex}
	sh.MetaTargets = []*Sheet{s}
	sh.Columns[1].Kind = valueKind
	sh.Columns[2].Kind = KindInt
	sh.Graph = &GraphSpec{
		Kind:   GraphBar,
		Title:  fmt.Sprintf("%s by %s", s.Columns[valueIndex].Name, s.Columns[labelIndex].Name),
		XLabel: s.Columns[labelIndex].Name,
		YLabel: s.Columns[valueIndex].Name,
	}

	x := 1.0
	for rowIndex := range s.Rows {
		value, ok := parseNumericValue(valueKind, s.Cell(rowIndex, valueIndex))
		if !ok {
			continue
		}
		label := s.Cell(rowIndex, labelIndex)
		sh.Graph.Points = append(sh.Graph.Points, GraphPoint{Label: label, X: x, Y: value})
		sh.AddRawRow([]string{
			label,
			formatNumericValue(valueKind, value),
			strconv.FormatFloat(x, 'f', 0, 64),
		})
		x++
	}

	if len(sh.Graph.Points) == 0 {
		return nil, fmt.Errorf("graph needs numeric values in %s", s.Columns[valueIndex].Name)
	}
	return sh, nil
}

func (s *Sheet) lineGraphSheet(valueIndex int) (*Sheet, error) {
	valueKind := s.Columns[valueIndex].EffectiveKind()
	headers := []string{"row", "x", s.Columns[valueIndex].Name}
	sh := New(fmt.Sprintf("graph:%s:%s", s.Name, s.Columns[valueIndex].Name), s.Source, headers)
	sh.MetaKind = "graph"
	sh.MetaCols = []int{valueIndex}
	sh.MetaTargets = []*Sheet{s}
	sh.Columns[0].Kind = KindInt
	sh.Columns[1].Kind = KindInt
	sh.Columns[2].Kind = valueKind
	sh.Graph = &GraphSpec{
		Kind:   GraphLine,
		Title:  fmt.Sprintf("%s over row order", s.Columns[valueIndex].Name),
		XLabel: "row",
		YLabel: s.Columns[valueIndex].Name,
	}

	for rowIndex := range s.Rows {
		value, ok := parseNumericValue(valueKind, s.Cell(rowIndex, valueIndex))
		if !ok {
			continue
		}
		x := float64(rowIndex + 1)
		label := strconv.Itoa(rowIndex + 1)
		sh.Graph.Points = append(sh.Graph.Points, GraphPoint{Label: label, X: x, Y: value})
		sh.AddRawRow([]string{
			label,
			strconv.Itoa(rowIndex + 1),
			formatNumericValue(valueKind, value),
		})
	}

	if len(sh.Graph.Points) == 0 {
		return nil, fmt.Errorf("graph needs numeric values in %s", s.Columns[valueIndex].Name)
	}
	return sh, nil
}

func (s *Sheet) scatterGraphSheet(xIndex, yIndex int) (*Sheet, error) {
	xKind := s.Columns[xIndex].EffectiveKind()
	yKind := s.Columns[yIndex].EffectiveKind()
	headers := []string{"label", s.Columns[xIndex].Name, s.Columns[yIndex].Name}
	sh := New(fmt.Sprintf("graph:%s:%s-vs-%s", s.Name, s.Columns[yIndex].Name, s.Columns[xIndex].Name), s.Source, headers)
	sh.MetaKind = "graph"
	sh.MetaCols = []int{xIndex, yIndex}
	sh.MetaTargets = []*Sheet{s}
	sh.Columns[1].Kind = xKind
	sh.Columns[2].Kind = yKind
	sh.Graph = &GraphSpec{
		Kind:   GraphScatter,
		Title:  fmt.Sprintf("%s vs %s", s.Columns[yIndex].Name, s.Columns[xIndex].Name),
		XLabel: s.Columns[xIndex].Name,
		YLabel: s.Columns[yIndex].Name,
	}

	for rowIndex := range s.Rows {
		xValue, okX := parseNumericValue(xKind, s.Cell(rowIndex, xIndex))
		yValue, okY := parseNumericValue(yKind, s.Cell(rowIndex, yIndex))
		if !okX || !okY {
			continue
		}
		label := strconv.Itoa(rowIndex + 1)
		sh.Graph.Points = append(sh.Graph.Points, GraphPoint{Label: label, X: xValue, Y: yValue})
		sh.AddRawRow([]string{
			label,
			formatNumericValue(xKind, xValue),
			formatNumericValue(yKind, yValue),
		})
	}

	if len(sh.Graph.Points) == 0 {
		return nil, fmt.Errorf("graph needs numeric values in %s and %s", s.Columns[xIndex].Name, s.Columns[yIndex].Name)
	}
	return sh, nil
}

func (s *Sheet) otherNumericColumn(columnIndex int) int {
	for _, visibleIndex := range s.VisibleColumnIndices() {
		if visibleIndex == columnIndex {
			continue
		}
		if isNumericKind(s.Columns[visibleIndex].EffectiveKind()) {
			return visibleIndex
		}
	}
	return -1
}

func (s *Sheet) firstNumericColumnExcluding(columnIndex int) int {
	return s.otherNumericColumn(columnIndex)
}

func isNumericKind(kind ValueKind) bool {
	return kind == KindInt || kind == KindFloat || kind == KindCurrency
}
