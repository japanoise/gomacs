package main

import (
	"bytes"
	"errors"
	"strings"

	"github.com/japanoise/termbox-util"
)

type Region struct {
	startc, endc int
	startl, endl int
}

func NewRegion(startc, endc, startl, endl int) *Region {
	return &Region{
		startc, endc,
		startl, endl,
	}
}

func (buf *EditorBuffer) setRegion(startc, startl, endc, endl int) {
	if buf.region == nil {
		buf.region = &Region{}
	}
	region := buf.region
	region.startl = startl
	if region.startl < buf.NumRows {
		region.startc = buf.Rows[region.startl].cxToRx(startc)
	} else {
		region.startc = 0
	}
	region.endl = endl
	if region.endl < buf.NumRows {
		region.endc = buf.Rows[region.endl].cxToRx(endc)
	} else {
		region.endc = 0
	}
}

func setMark(buf *EditorBuffer) {
	if buf.cx == buf.MarkX && buf.cy == buf.MarkY {
		if buf.regionActive == false {
			Global.Input = "Mark activated."
		} else {
			Global.Input = "Mark deactivated."
			buf.regionActive = false
			return
		}
	} else {
		buf.MarkX = buf.cx
		buf.MarkY = buf.cy
		Global.Input = "Mark set."
	}
	buf.regionActive = true
	buf.recalcRegion()
}

func (buf *EditorBuffer) recalcRegion() {
	if markAhead(buf) {
		buf.setRegion(buf.cx, buf.cy, buf.MarkX, buf.MarkY)
	} else {
		buf.setRegion(buf.MarkX, buf.MarkY, buf.cx, buf.cy)
	}
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

func rowDelRange(row *EditorRow, startc, endc int, buf *EditorBuffer) string {
	editorAddDeleteUndo(startc, endc,
		row.idx, row.idx, row.Data[startc:endc])
	ret := row.Data[startc:endc]
	editorRowDelChar(row, buf, startc, endc-startc)
	return ret
}

func bufKillRegion(buf *EditorBuffer, startc, endc, startl, endl int) string {
	var ret string
	row := buf.Rows[startl]
	if startl == endl {
		ret = row.Data[startc:endc]
		editorRowDelChar(row, buf, startc, endc-startc)
	} else {
		var bb bytes.Buffer

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
		ret = bb.String()

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
	}
	buf.cx = startc
	buf.prefcx = startc
	buf.cy = startl
	buf.Dirty = true
	return ret
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

func bufCopyRegion(buf *EditorBuffer, startc, endc, startl, endl int) string {
	return getRegionText(buf, startc, endc, startl, endl)
}

func markAhead(buf *EditorBuffer) bool {
	if buf.MarkY == buf.cy {
		return buf.MarkX > buf.cx
	} else {
		return buf.MarkY > buf.cy
	}
}

func regionCmd(c func(*EditorBuffer, int, int, int, int) string) (string, error) {
	buf := Global.CurrentB
	if !validMark(buf) {
		Global.Input = "Invalid mark position"
		return "", errors.New("invalid mark")
	}
	if markAhead(buf) {
		return c(buf, buf.cx, buf.MarkX, buf.cy, buf.MarkY), nil
	} else {
		return c(buf, buf.MarkX, buf.cx, buf.MarkY, buf.cy), nil
	}
}

func doKillRegion() {
	res, err := regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) string {
		ret := bufKillRegion(buf, startc, endc, startl, endl)
		editorAddRegionUndo(false, startc, endc,
			startl, endl, Global.Clipboard)
		return ret
	})
	if err == nil {
		Global.Clipboard = res
		Global.CurrentB.regionActive = false
	}
}

func doCopyRegion() {
	res, err := regionCmd(bufCopyRegion)
	if err == nil {
		Global.Clipboard = res
		Global.CurrentB.regionActive = false
	}
}

func spitRegion(cx, cy int, region string) (int, int) {
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
	return cx, cy
}

func doYankText(text string) {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		cx, cy := spitRegion(Global.CurrentB.cx, Global.CurrentB.cy, text)
		editorAddRegionUndo(true, cx, Global.CurrentB.cx,
			cy, Global.CurrentB.cy, text)
	}
}

func doYankRegion() {
	doYankText(Global.Clipboard)
	Global.CurrentB.regionActive = false
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
				Global.Clipboard = rowDelRange(Global.CurrentB.Rows[cy], 0, cx, Global.CurrentB)
				Global.CurrentB.cx = 0
			}
		} else if 1 < Global.Universal {
			endl := cy + Global.Universal
			if Global.CurrentB.NumRows < endl {
				endl = Global.CurrentB.NumRows - 1
			}
			Global.Clipboard = bufKillRegion(Global.CurrentB, cx, 0, cy, endl)
			editorAddRegionUndo(false, cx, 0, cy, endl, Global.Clipboard)
		} else {
			startl := cy + Global.Universal
			if startl < 0 {
				startl = 0
			}
			Global.Clipboard = bufKillRegion(Global.CurrentB, 0, cx, startl, cy)
			editorAddRegionUndo(false, 0, cx, startl, cy, Global.Clipboard)
		}
	} else {
		if cx >= Global.CurrentB.Rows[cy].Size {
			Global.CurrentB.MoveCursorRight()
			editorDelChar()
		} else {
			Global.Clipboard = rowDelRange(Global.CurrentB.Rows[cy], cx, Global.CurrentB.Rows[cy].Size, Global.CurrentB)
		}
	}
}

func transposeRegion(buf *EditorBuffer, startc, endc, startl, endl int, trans func(string) string) {
	killed := bufKillRegion(buf, startc, endc, startl, endl)
	editorAddRegionUndo(false, startc, endc, startl, endl, killed)
	text := trans(killed)
	cx, cy := spitRegion(startc, startl, text)
	editorAddRegionUndo(true, cx, buf.cx, cy, buf.cy, text)
	buf.Undo.paired = true
}

func transposeRegionCmd(trans func(string) string) {
	regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) string {
		transposeRegion(buf, startc, endc, startl, endl, trans)
		return ""
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
	ret := bytes.Buffer{}
	lw := 0
	for _, line := range lines {
		if line == "" {
			if lw != 0 {
				ret.WriteString("\n")
			}
			ret.WriteString("\n")
			lw = 0
			continue
		}
		chomp := line
		for chomp != "" {
			var word string
			chomp, word = chompWord(chomp)
			if word == "" {
				break
			}
			ww := termutil.RunewidthStr(word)
			if lw+ww > Global.Fillcolumn {
				ret.WriteString("\n" + word)
				lw = ww
			} else {
				if lw == 0 {
					ret.WriteString(word)
				} else {
					ret.WriteString(" " + word)
					lw++
				}
				lw += ww
			}
		}
	}
	return ret.String()
}

func chompWord(s string) (string, string) {
	if s == "" {
		return "", ""
	} else {
		i := chompIndex(s)
		if i == -1 {
			return "", ""
		}
		return s[i:], strings.Trim(s[:i], " ")
	}
}

func chompIndex(s string) int {
	leader := true
	for i, ru := range s {
		if ru == ' ' {
			if !leader {
				return i
			}
		} else {
			leader = false
		}
	}
	return len(s)
}

func doFillRegion() {
	transposeRegionCmd(FillString)
}
