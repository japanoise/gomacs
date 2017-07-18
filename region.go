package main

import (
	"bytes"
	"strings"

	"github.com/japanoise/termbox-util"
)

func setMark(buf *EditorBuffer) {
	buf.MarkX = buf.cx
	buf.MarkY = buf.cy
	Global.Input = "Mark set."
}

func validMark(buf *EditorBuffer) bool {
	return buf.cy < buf.NumRows && buf.MarkY < buf.NumRows && buf.MarkX <= len(buf.Rows[buf.MarkY].Data)
}

func doSwapMarkAndCursor(buf *EditorBuffer) {
	if validMark(buf) {
		cx, cy := buf.cx, buf.cy
		buf.cx = buf.MarkX
		buf.cy = buf.MarkY
		buf.MarkX = cx
		buf.MarkY = cy
	} else {
		Global.Input = "Invalid mark position"
	}
}

func rowDelRange(row *EditorRow, startc, endc int, buf *EditorBuffer) {
	editorAddDeleteUndo(startc, endc,
		row.idx, row.idx, row.Data[startc:endc])
	Global.Clipboard = row.Data[startc:endc]
	editorRowDelChar(row, buf, startc, endc-startc)
}

func bufKillRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	if startl == endl {
		rowDelRange(buf.Rows[startl], startc, endc, buf)
	} else {
		var bb bytes.Buffer
		row := buf.Rows[startl]

		// Delete from first line
		bb.WriteString(row.Data[startc:])
		row.Data = row.Data[:startc]
		bb.WriteRune('\n')

		// Collect data from middle rows
		for i := startl + 1; i < endl; i++ {
			bb.WriteString(buf.Rows[i].Data)
			bb.WriteRune('\n')
		}

		// Collect data from last row
		row = buf.Rows[endl]
		bb.WriteString(row.Data[:endc])
		row.Data = row.Data[endc:]

		// Append last row's data to first row
		buf.Rows[startl].Data += row.Data
		buf.Rows[startl].Size = len(buf.Rows[startl].Data)
		rowUpdateRender(buf.Rows[startl])
		Global.Clipboard = bb.String()

		// Cut region out of rows
		i, j := startl+1, endl+1
		copy(buf.Rows[i:], buf.Rows[j:])
		for k, n := len(buf.Rows)-j+i, len(buf.Rows); k < n; k++ {
			buf.Rows[k] = nil // or the zero value of T
		}
		buf.Rows = buf.Rows[:len(buf.Rows)-j+i]
		buf.NumRows = len(buf.Rows)

		// Update the buffer and return
		updateLineIndexes()
		buf.Highlight()
		editorAddRegionUndo(false, startc, endc,
			startl, endl, Global.Clipboard)
	}
	buf.cx = startc
	buf.prefcx = startc
	buf.cy = startl
	buf.Dirty = true
}

func getRegionText(buf *EditorBuffer, startc, endc, startl, endl int) string {
	if startl == endl {
		return buf.Rows[startl].Data[startc:endc]
	} else {
		var bb bytes.Buffer
		row := buf.Rows[startl]
		bb.WriteString(row.Data[startc:])
		bb.WriteRune('\n')
		for i := startl + 1; i < endl; i++ {
			row = buf.Rows[i]
			bb.WriteString(row.Data)
			bb.WriteRune('\n')
		}
		row = buf.Rows[endl]
		bb.WriteString(row.Data[:endc])
		return bb.String()
	}
}

func bufCopyRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	Global.Clipboard = getRegionText(buf, startc, endc, startl, endl)
}

func markAhead(buf *EditorBuffer) bool {
	if buf.MarkY == buf.cy {
		return buf.MarkX > buf.cx
	} else {
		return buf.MarkY > buf.cy
	}
}

func regionCmd(c func(*EditorBuffer, int, int, int, int)) {
	buf := Global.CurrentB
	if !validMark(buf) {
		Global.Input = "Invalid mark position"
		return
	}
	if markAhead(buf) {
		c(buf, buf.cx, buf.MarkX, buf.cy, buf.MarkY)
	} else {
		c(buf, buf.MarkX, buf.cx, buf.MarkY, buf.cy)
	}
}

func doKillRegion() {
	regionCmd(bufKillRegion)
}

func doCopyRegion() {
	regionCmd(bufCopyRegion)
}

func spitRegion(cx, cy int, region string) {
	Global.CurrentB.Dirty = true
	Global.CurrentB.cx = cx
	Global.CurrentB.prefcx = cx
	Global.CurrentB.cy = cy
	clipLines := strings.Split(region, "\n")
	if cy == Global.CurrentB.NumRows {
		editorAppendRow("")
	}
	row := Global.CurrentB.Rows[cy]
	data := row.Data
	row.Data = data[:cx] + clipLines[0]
	row.Size = len(row.Data)
	Global.CurrentB.cx = row.Size
	Global.CurrentB.prefcx = row.Size
	if len(clipLines) > 1 {
		// Insert more lines...
		rowUpdateRender(row)
		myrows := make([]*EditorRow, len(clipLines)-1)
		mrlen := len(myrows)
		for i := 0; i < mrlen; i++ {
			newrow := &EditorRow{}
			newrow.Data = clipLines[i+1]
			newrow.Size = len(newrow.Data)
			rowUpdateRender(newrow)
			myrows[i] = newrow
		}
		Global.CurrentB.cy += mrlen
		Global.CurrentB.cx = myrows[mrlen-1].Size
		Global.CurrentB.prefcx = Global.CurrentB.cx
		if cx < len(data) {
			myrows[mrlen-1].Data += data[cx:]
			myrows[mrlen-1].Size = len(myrows[mrlen-1].Data)
			rowUpdateRender(myrows[mrlen-1])
		}

		if cy < Global.CurrentB.NumRows {
			Global.CurrentB.Rows = append(Global.CurrentB.Rows[:cy+1], append(myrows, Global.CurrentB.Rows[cy+1:]...)...)

		} else {
			Global.CurrentB.Rows = append(Global.CurrentB.Rows[:cy], myrows...)
		}
		Global.CurrentB.NumRows = len(Global.CurrentB.Rows)
		updateLineIndexes()
		if Global.CurrentB.Highlighter != nil {
			Global.CurrentB.Highlighter.HighlightStates(Global.CurrentB)
			if cy == 0 {
				Global.CurrentB.Highlighter.HighlightMatches(Global.CurrentB, 0, Global.CurrentB.NumRows)

			} else {
				Global.CurrentB.Highlighter.HighlightMatches(Global.CurrentB, cy-1, Global.CurrentB.NumRows)

			}
		}
	} else {
		row.Data += data[cx:]
		row.Size = len(row.Data)
		editorUpdateRow(row, Global.CurrentB)
	}
	editorAddRegionUndo(true, cx, Global.CurrentB.cx,
		cy, Global.CurrentB.cy, region)
}

func doYankText(text string) {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		spitRegion(Global.CurrentB.cx, Global.CurrentB.cy, text)
	}
}

func doYankRegion() {
	doYankText(Global.Clipboard)
}

func killToEol() {
	cx := Global.CurrentB.cx
	cy := Global.CurrentB.cy
	if cy == Global.CurrentB.NumRows {
		return
	}
	if Global.SetUniversal && Global.Universal != 1 {
		if Global.Universal == 0 {
			if 0 < Global.CurrentB.cx && cy < Global.CurrentB.NumRows {
				rowDelRange(Global.CurrentB.Rows[cy], 0, cx, Global.CurrentB)
				Global.CurrentB.cx = 0
			}
		} else if 1 < Global.Universal {
			endl := cy + Global.Universal
			if Global.CurrentB.NumRows < endl {
				endl = Global.CurrentB.NumRows - 1
			}
			bufKillRegion(Global.CurrentB, cx, 0, cy, endl)
		} else {
			startl := cy + Global.Universal
			if startl < 0 {
				startl = 0
			}
			bufKillRegion(Global.CurrentB, 0, cx, startl, cy)
		}
	} else {
		if cx >= Global.CurrentB.Rows[cy].Size {
			Global.CurrentB.MoveCursorRight()
			editorDelChar()
		} else {
			rowDelRange(Global.CurrentB.Rows[cy], cx, Global.CurrentB.Rows[cy].Size, Global.CurrentB)
		}
	}
}

func transposeRegion(buf *EditorBuffer, startc, endc, startl, endl int, trans func(string) string) {
	clip := Global.Clipboard
	bufKillRegion(buf, startc, endc, startl, endl)
	spitRegion(startc, startl, trans(Global.Clipboard))
	Global.Clipboard = clip
}

func transposeRegionCmd(trans func(string) string) {
	regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) {
		transposeRegion(buf, startc, endc, startl, endl, trans)
	})
}

func doUCRegion() {
	transposeRegionCmd(strings.ToUpper)
}

func doLCRegion() {
	transposeRegionCmd(strings.ToLower)
}

func doUntabifyRegion() {
	transposeRegionCmd(func(s string) string {
		lines := strings.Split(s, "\n")
		newlines := make([]string, 0, len(lines))
		repstr := ""
		for i := 0; i < Global.Tabsize; i++ {
			repstr += " "
		}
		for _, line := range lines {
			if strings.HasPrefix(line, "\t") {
				count := 0
				for _, ru := range line {
					if ru == '\t' {
						count++
					} else {
						break
					}
				}
				newlines = append(newlines, strings.Replace(line, "\t", repstr, count))
			} else {
				newlines = append(newlines, line)
			}
		}
		return strings.Join(newlines, "\n")
	})
}

func doTabifyRegion() {
	transposeRegionCmd(func(s string) string {
		lines := strings.Split(s, "\n")
		newlines := make([]string, 0, len(lines))
		repstr := ""
		for i := 0; i < Global.Tabsize; i++ {
			repstr += " "
		}
		for _, line := range lines {
			if strings.HasPrefix(line, " ") {
				count := 0
				for _, ru := range line {
					if ru == ' ' {
						count++
					} else {
						break
					}
				}
				count = count / Global.Tabsize
				newlines = append(newlines, strings.Replace(line, repstr, "\t", count))
			} else {
				newlines = append(newlines, line)
			}
		}
		return strings.Join(newlines, "\n")
	})
}

func FillString(s string) string {
	lines := strings.Split(s, "\n")
	ll := len(lines)
	for i := 0; i < ll; i++ {
		line := lines[i] // We can't use range here as we may be modifying lines
		linelen := termutil.RunewidthStr(line)
		if linelen > Global.Fillcolumn {
			var index int
			rl := []rune(line)
			fill := rl[Global.Fillcolumn]
			afterfill := rl[Global.Fillcolumn+1]
			if (termutil.WordCharacter(fill) && !termutil.WordCharacter(afterfill)) ||
				!termutil.WordCharacter(fill) {
				index = Global.Fillcolumn
			} else {
				index = indexOfLastWord(line[:Global.Fillcolumn])
			}
			if index > 0 {
				if i == ll-1 {
					lines = append(lines, "")
					ll++
				} else if lines[i+1] == "" {
					n := i + 1
					lines = append(lines, "")
					copy(lines[n+1:], lines[n:])
					lines[n] = ""
					ll++
				}
				lines[i+1] = line[index:] + lines[i+1]
				lines[i] = line[:index]
			}
		} else if i != ll-1 && line != "" {
			index := indexOfFirstWord(lines[i+1])
			wi := index
			working := true
			for working {
				oldwi := wi
				wi = wi + indexOfFirstWord(lines[i+1][wi:])
				if oldwi == wi {
					working = false
				}
				if termutil.RunewidthStr(lines[i+1][:wi]) < Global.Fillcolumn-linelen {
					index = wi
				} else {
					working = false
				}
			}
			if index < Global.Fillcolumn-linelen && index > 0 {
				lines[i] = line + lines[i+1][:index]
				lines[i+1] = lines[i+1][index:]
			}
		}
	}
	return strings.Join(lines, "\n")
}

func doFillRegion() {
	transposeRegionCmd(FillString)
}
