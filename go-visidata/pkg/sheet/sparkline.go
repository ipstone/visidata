package sheet

import (
	"fmt"
	"math"
)

var sparklineRunes = []rune("▁▂▃▄▅▆▇█")

func (s *Sheet) AddSparklineColumn() (string, error) {
	numericCols := s.visibleNumericColumnIndices()
	if len(numericCols) == 0 {
		return "", fmt.Errorf("sparkline needs at least one visible numeric column")
	}

	values := make([]string, len(s.Rows))
	for rowIndex := range s.Rows {
		rowValues := make([]float64, 0, len(numericCols))
		for _, colIndex := range numericCols {
			value, ok := parseNumericValue(s.Columns[colIndex].EffectiveKind(), s.Cell(rowIndex, colIndex))
			if ok {
				rowValues = append(rowValues, value)
			}
		}
		values[rowIndex] = sparkline(rowValues)
	}

	name := uniqueColumnName(s, "sparkline")
	s.pushUndo("add sparkline column")
	s.Columns = append(s.Columns, Column{Name: name, Kind: KindString})
	for rowIndex := range s.Rows {
		s.Rows[rowIndex] = append(s.Rows[rowIndex], values[rowIndex])
	}
	s.CursorCol = len(s.Columns) - 1
	s.clampCursor()
	s.refreshSearch()
	return name, nil
}

func (s *Sheet) visibleNumericColumnIndices() []int {
	indices := make([]int, 0)
	for _, colIndex := range s.VisibleColumnIndices() {
		if isNumericKind(s.Columns[colIndex].EffectiveKind()) {
			indices = append(indices, colIndex)
		}
	}
	return indices
}

func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return string(sparklineRunes[len(sparklineRunes)/2])
	}

	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}
	if maxValue == minValue {
		return repeatSparkline(len(values), sparklineRunes[len(sparklineRunes)/2])
	}

	out := make([]rune, len(values))
	maxIndex := float64(len(sparklineRunes) - 1)
	for i, value := range values {
		scaled := (value - minValue) / (maxValue - minValue)
		index := int(math.Round(scaled * maxIndex))
		if index < 0 {
			index = 0
		}
		if index >= len(sparklineRunes) {
			index = len(sparklineRunes) - 1
		}
		out[i] = sparklineRunes[index]
	}
	return string(out)
}

func repeatSparkline(count int, r rune) string {
	out := make([]rune, count)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
