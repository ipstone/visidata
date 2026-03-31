package tui

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func (a *App) drawGraph(x0, width, height int, bodyStyle, headerStyle tcell.Style) {
	spec := a.Sheet.Graph
	if spec == nil || height < 4 || width < 8 {
		return
	}

	title := fmt.Sprintf(" %s [%s] ", spec.Title, spec.Kind)
	a.drawText(x0, 1, width, title, headerStyle)

	plotTop := 2
	plotHeight := height - plotTop - 1
	if plotHeight < 2 {
		return
	}

	points := spec.Points
	if len(points) == 0 {
		a.drawText(x0, plotTop, width, "no graph points", bodyStyle)
		return
	}

	minY, maxY := points[0].Y, points[0].Y
	minX, maxX := points[0].X, points[0].X
	for _, point := range points[1:] {
		minY = math.Min(minY, point.Y)
		maxY = math.Max(maxY, point.Y)
		minX = math.Min(minX, point.X)
		maxX = math.Max(maxX, point.X)
	}
	if spec.Kind == sheet.GraphBar {
		minY = math.Min(minY, 0)
		maxY = math.Max(maxY, 0)
	}
	if maxY == minY {
		maxY++
	}
	if maxX == minX {
		maxX++
	}

	leftPad := min(12, max(8, max(displayWidth(formatAxisValue(maxY)), displayWidth(formatAxisValue(minY)))+2))
	plotWidth := width - leftPad
	dataHeight := plotHeight - 1
	if plotWidth < 2 || dataHeight < 1 {
		return
	}

	a.drawText(x0, plotTop, leftPad-1, padRight(formatAxisValue(maxY), leftPad-1), bodyStyle)
	a.drawText(x0, plotTop+dataHeight-1, leftPad-1, padRight(formatAxisValue(minY), leftPad-1), bodyStyle)

	grid := make([][]rune, dataHeight)
	for y := range grid {
		grid[y] = make([]rune, plotWidth)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	if spec.Kind == sheet.GraphBar {
		for index, point := range points {
			x := scaledBarX(index, len(points), plotWidth)
			y := scaledY(point.Y, minY, maxY, dataHeight)
			for row := y; row < dataHeight; row++ {
				grid[row][x] = '#'
			}
		}
	} else {
		prevX, prevY := -1, -1
		for index, point := range points {
			x := scaledX(point.X, minX, maxX, plotWidth)
			y := scaledY(point.Y, minY, maxY, dataHeight)
			mark := '*'
			if index == a.Sheet.CursorRow {
				mark = '@'
			}
			if spec.Kind == sheet.GraphLine && prevX >= 0 && prevY >= 0 {
				drawLine(grid, prevX, prevY, x, y, '.')
			}
			grid[y][x] = mark
			prevX, prevY = x, y
		}
	}

	if a.Sheet.CursorRow >= 0 && a.Sheet.CursorRow < len(points) && spec.Kind == sheet.GraphBar {
		point := points[a.Sheet.CursorRow]
		x := scaledBarX(a.Sheet.CursorRow, len(points), plotWidth)
		y := scaledY(point.Y, minY, maxY, dataHeight)
		grid[y][x] = '@'
	}

	style := bodyStyle
	if spec.Kind == sheet.GraphScatter {
		style = style.Foreground(a.theme.AccentFG)
	}
	for y := 0; y < dataHeight; y++ {
		a.drawText(x0+leftPad, plotTop+y, plotWidth, string(grid[y]), style)
	}

	xLabelLine := fmt.Sprintf("%s [%s .. %s]", spec.XLabel, formatAxisValue(minX), formatAxisValue(maxX))
	if spec.Kind == sheet.GraphBar && len(points) > 0 {
		xLabelLine = fmt.Sprintf("%s [%s .. %s]", spec.XLabel, points[0].Label, points[len(points)-1].Label)
	}
	a.drawText(x0+leftPad, plotTop+dataHeight, plotWidth, padRight(xLabelLine, plotWidth), bodyStyle)
}

func scaledX(value, minX, maxX float64, width int) int {
	if width <= 1 || maxX == minX {
		return 0
	}
	position := (value - minX) / (maxX - minX)
	return max(0, min(width-1, int(math.Round(position*float64(width-1)))))
}

func scaledBarX(index, count, width int) int {
	if width <= 1 || count <= 1 {
		return 0
	}
	position := float64(index) / float64(count-1)
	return max(0, min(width-1, int(math.Round(position*float64(width-1)))))
}

func scaledY(value, minY, maxY float64, height int) int {
	if height <= 1 || maxY == minY {
		return 0
	}
	position := (value - minY) / (maxY - minY)
	return max(0, min(height-1, height-1-int(math.Round(position*float64(height-1)))))
}

func drawLine(grid [][]rune, x0, y0, x1, y1 int, mark rune) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy

	for {
		if y0 >= 0 && y0 < len(grid) && x0 >= 0 && x0 < len(grid[y0]) && grid[y0][x0] == ' ' {
			grid[y0][x0] = mark
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func formatAxisValue(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}
