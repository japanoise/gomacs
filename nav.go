package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func editorScroll(sx, sy int) {
	Global.CurrentB.rx = 0
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.rx = editorRowCxToRx(Global.CurrentB.Rows[Global.CurrentB.cy])
	}

	if Global.CurrentB.cy < Global.CurrentB.rowoff {
		Global.CurrentB.rowoff = Global.CurrentB.cy
	}
	if Global.CurrentB.cy >= Global.CurrentB.rowoff+sy {
		Global.CurrentB.rowoff = Global.CurrentB.cy - sy + 1
	}
	if Global.CurrentB.rx < Global.CurrentB.coloff {
		Global.CurrentB.coloff = Global.CurrentB.rx
	}
	if Global.CurrentB.rx >= Global.CurrentB.coloff+sx {
		Global.CurrentB.coloff = Global.CurrentB.rx - sx + 1
	}
}

func editorCentreView() {
	rowoff := Global.CurrentB.cy - (Global.CurrentBHeight / 2)
	if rowoff >= 0 {
		Global.CurrentB.rowoff = rowoff
	}
}

func (buf *EditorBuffer) UpdateRowToPrefCX() {
	row := buf.Rows[buf.cy]
	if buf.prefcx == -1 || buf.prefcx > row.Size {
		buf.cx = row.Size
	} else {
		buf.cx = buf.prefcx
	}
}

func (buf *EditorBuffer) MoveCursorToEndOfBuffer() {
	buf.cy = len(buf.Rows) - 1
	buf.cx = buf.Rows[buf.cy].Size
	buf.prefcx = -1
}

func (buf *EditorBuffer) MoveCursorDown() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		if buf.cy >= buf.NumRows-1 {
			Global.Input = "End of buffer"
		} else {
			buf.cy++
			buf.UpdateRowToPrefCX()
		}
	}
}

func (buf *EditorBuffer) MoveCursorUp() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		if buf.cy == 0 {
			Global.Input = "Beginning of buffer"
		} else {
			buf.cy--
			buf.UpdateRowToPrefCX()
		}
	}
}

func (buf *EditorBuffer) MoveCursorLeft() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		if buf.cy == 0 && buf.cx == 0 {
			Global.Input = "Beginning of buffer"
		} else if buf.cx == 0 {
			buf.cy--
			buf.prefcx = -1
			buf.cx = buf.Rows[buf.cy].Size
		} else {
			_, rs :=
				utf8.DecodeLastRuneInString(buf.Rows[buf.cy].Data[:buf.cx])
			buf.cx -= rs
			buf.prefcx = buf.cx
		}
	}
}

func (buf *EditorBuffer) MoveCursorRight() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		if buf.cy >= buf.NumRows {
			Global.Input = "End of buffer"
		} else if buf.cx == buf.Rows[buf.cy].Size {
			if buf.cy == buf.NumRows-1 {
				Global.Input = "End of buffer"
			} else {
				buf.cy++
				buf.prefcx = 0
				buf.cx = 0
			}
		} else {
			_, rs := utf8.DecodeRuneInString(buf.Rows[buf.cy].Data[buf.cx:])
			buf.cx += rs
			buf.prefcx = buf.cx
		}
	}
}

func MoveCursorToEol() {
	Global.CurrentB.prefcx = -1
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
	}
}

func MoveCursorToBol() {
	Global.CurrentB.cx = 0
	Global.CurrentB.prefcx = 0
}

func MovePage(back bool, sy int) {
	for i := 0; i < sy; i++ {
		if back {
			Global.CurrentB.MoveCursorUp()
		} else {
			Global.CurrentB.MoveCursorDown()
		}
	}
}

func MoveCursorBackPage() {
	if Global.SetUniversal {
		sy := Global.Universal
		if sy < 0 {
			Global.Universal *= -1
			MoveCursorForthPage()
			return
		} else if Global.CurrentB.rowoff-sy >= 0 {
			Global.CurrentB.rowoff -= sy
		} else {
			Global.Input = "Beginning of buffer"
			Global.CurrentB.rowoff = 0
		}
		_, ssy := GetScreenSize()
		for Global.CurrentB.cy > Global.CurrentB.rowoff+ssy {
			Global.CurrentB.MoveCursorUp()
		}
	} else {
		_, sy := GetScreenSize()
		Global.CurrentB.cy = Global.CurrentB.rowoff
		MovePage(true, sy)
	}
}

func MoveCursorForthPage() {
	if Global.SetUniversal {
		sy := Global.Universal
		if sy < 0 {
			Global.Universal *= -1
			MoveCursorBackPage()
		} else if Global.CurrentB.rowoff+sy < Global.CurrentB.NumRows {
			Global.CurrentB.rowoff += sy
			for Global.CurrentB.cy < Global.CurrentB.rowoff {
				Global.CurrentB.MoveCursorDown()
			}
		} else {
			Global.Input = "End of buffer"
		}
	} else {
		_, sy := GetScreenSize()
		Global.CurrentB.cy = Global.CurrentB.rowoff + sy - 1
		if Global.CurrentB.cy > Global.CurrentB.NumRows {
			Global.CurrentB.cy = Global.CurrentB.NumRows
		}
		MovePage(false, sy)
	}
}

// HACK: Go does not have static variables, so these have to go in global state.
var last_match int = -1
var direction int = 1

func editorFindCallback(query string, key string) {
	Global.Input = query
	if key == "C-s" {
		direction = 1
	} else if key == "C-r" {
		direction = -1
		//If we cancelled or finished...
	} else if key == "C-c" || key == "C-g" || key == "RET" {
		if key == "C-c" || key == "C-g" {
			Global.Input = "Cancelled search."
		}
		//...outta here!
		last_match = -1
		direction = 1
		Global.CurrentB.regionActive = false
		return
	} else {
		last_match = -1
		direction = 1
	}

	if last_match == -1 {
		direction = 1
	}
	current := last_match
	ql := len(query)
	for range Global.CurrentB.Rows {
		current += direction
		if current == -1 {
			current = Global.CurrentB.NumRows - 1
		} else if current == Global.CurrentB.NumRows {
			current = 0
		}
		row := Global.CurrentB.Rows[current]
		match := strings.Index(row.Render, query)
		if match > -1 {
			last_match = current
			Global.CurrentB.cy = current
			Global.CurrentB.cx = editorRowRxToCx(row, match)
			Global.CurrentB.prefcx = Global.CurrentB.cx
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			Global.CurrentB.MarkX = Global.CurrentB.cx + ql
			Global.CurrentB.MarkY = Global.CurrentB.cy
			Global.CurrentB.regionActive = true
			Global.CurrentB.recalcRegion()
			break
		}
	}
}

func editorFind() {
	saved_cx := Global.CurrentB.cx
	saved_cy := Global.CurrentB.cy
	saved_co := Global.CurrentB.coloff
	saved_ro := Global.CurrentB.rowoff

	query := editorPrompt("Search", editorFindCallback)

	if query == "" {
		//Search cancelled, go back to where we were
		Global.CurrentB.regionActive = false
		Global.CurrentB.cx = saved_cx
		Global.CurrentB.prefcx = Global.CurrentB.cx
		Global.CurrentB.cy = saved_cy
		Global.CurrentB.coloff = saved_co
		Global.CurrentB.rowoff = saved_ro
	}
}

func doQueryReplace() {
	orig := editorPrompt("Find", nil)
	if orig == "" {
		Global.Input = "Can't query-replace with an empty query"
		return
	}
	defer func() { Global.CurrentB.regionActive = false }()
	replace := editorPrompt("Replace "+orig+" with", nil)
	all := false
	ql := len(orig)
	rlen := len(replace)
	for cy, row := range Global.CurrentB.Rows {
		match := strings.Index(row.Data, orig)
		prestring := ""
		matchstring := row.Data
		for match != -1 {
			psl := len(prestring)
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = match + psl
			Global.CurrentB.prefcx = Global.CurrentB.cx
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			Global.CurrentB.MarkX = Global.CurrentB.cx + ql
			Global.CurrentB.MarkY = Global.CurrentB.cy
			Global.CurrentB.regionActive = true
			Global.CurrentB.recalcRegion()

			var pressed string
			if !all {
				pressed = editorPressKey("Replace with "+replace+"?", "y", "n", "C-g", "q", ".", "!")
				if pressed == "!" {
					all = true
				}
			}

			if pressed == "C-g" || pressed == "q" {
				return
			} else if pressed == "y" || pressed == "." || all {
				Global.CurrentB.Dirty = true
				editorAddDeleteUndo(0, row.Size, cy, cy, row.Data)
				row.Data = prestring + strings.Replace(matchstring, orig, replace, 1)
				row.Size = len(row.Data)
				prestring = row.Data[:psl+match+rlen]
				matchstring = row.Data[psl+match+rlen:]
				editorAddInsertUndo(0, cy, row.Data)
				editorUpdateRow(row, Global.CurrentB)
				if pressed == "." {
					return
				}
			} else {
				prestring = row.Data[:psl+match+ql]
				matchstring = row.Data[psl+match+ql:]
			}
			match = strings.Index(matchstring, orig)
		}
	}
}

func doReplaceString() {
	orig := editorPrompt("Find", nil)
	if orig == "" {
		Global.Input = "Can't string-replace with an empty query"
		return
	}
	replace := editorPrompt("Replace "+orig+" with", nil)
	matches := 0
	lines := 0
	ql := len(orig)
	nl := len(replace)
	for cy, row := range Global.CurrentB.Rows {
		match := strings.LastIndex(row.Render, orig)
		if match != -1 {
			count := strings.Count(row.Render, orig)
			matches += count
			lines++
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = editorRowRxToCx(row, match+ql-(count*(ql-nl)))
			Global.CurrentB.prefcx = Global.CurrentB.cx
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			Global.CurrentB.Dirty = true
			editorAddDeleteUndo(0, row.Size, cy, cy, row.Data)
			row.Data = strings.Replace(row.Data, orig, replace, -1)
			row.Size = len(row.Data)
			editorAddInsertUndo(0, cy, row.Data)
			editorUpdateRow(row, Global.CurrentB)
		}
	}
	if matches > 0 {
		Global.Input = fmt.Sprintf("Replaced %d occurences on %d lines",
			matches, lines)
	} else {
		Global.Input = "No matches found"
	}
}

func doQueryReplaceRegexp() {
	orig := editorPrompt("Find regexp", nil)
	if orig == "" {
		Global.Input = "Can't query-replace-regexp with an empty query"
		return
	}
	defer func() { Global.CurrentB.regionActive = false }()
	pattern, err := regexp.Compile(orig)
	if err != nil {
		Global.Input = "Couldn't compile regexp " + orig + ": " + err.Error()
		return
	}
	replace := editorPrompt("Replace "+orig+" with", nil)
	all := false
	for cy, row := range Global.CurrentB.Rows {
		match := pattern.FindStringIndex(row.Data)
		prestring := ""
		matchstring := row.Data
		for len(match) != 0 {
			psl := len(prestring)
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = match[0] + psl
			Global.CurrentB.prefcx = Global.CurrentB.cx
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			matchlen := len(matchstring[match[0]:match[1]])
			Global.CurrentB.MarkX = Global.CurrentB.cx + matchlen
			Global.CurrentB.MarkY = Global.CurrentB.cy
			Global.CurrentB.regionActive = true
			Global.CurrentB.recalcRegion()

			var pressed string
			if !all {
				pressed = editorPressKey("Replace with "+replace+"?", "y", "n", "C-g", "q", ".", "!")
				if pressed == "!" {
					all = true
				}
			}

			if pressed == "C-g" || pressed == "q" {
				return
			} else if pressed == "y" || pressed == "." || all {
				Global.CurrentB.Dirty = true
				editorAddDeleteUndo(0, row.Size, cy, cy, row.Data)
				prestring = prestring + pattern.ReplaceAllString(matchstring[:match[1]], replace)
				matchstring = matchstring[match[1]:]
				row.Data = prestring + matchstring
				row.Size = len(row.Data)
				editorAddInsertUndo(0, cy, row.Data)
				editorUpdateRow(row, Global.CurrentB)
				if pressed == "." {
					return
				}
			} else {
				prestring = prestring + matchstring[:match[1]]
				matchstring = matchstring[match[1]:]
			}
			match = pattern.FindStringIndex(matchstring)
		}
	}
}

func doReplaceRegexp() {
	orig := editorPrompt("Find regexp", nil)
	if orig == "" {
		Global.Input = "Can't replace-regexp with an empty query"
		return
	}
	pattern, err := regexp.Compile(orig)
	if err != nil {
		Global.Input = "Couldn't compile regexp " + orig + ": " + err.Error()
		return
	}
	replace := editorPrompt("Replace "+orig+" with", nil)
	matches := 0
	lines := 0
	for cy, row := range Global.CurrentB.Rows {
		match := pattern.MatchString(row.Data)
		if match {
			count := len(pattern.FindAllString(row.Data, -1))
			matches += count
			lines++
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = editorRowRxToCx(row, row.Size)
			Global.CurrentB.prefcx = Global.CurrentB.cx
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			Global.CurrentB.Dirty = true
			editorAddDeleteUndo(0, row.Size, cy, cy, row.Data)
			row.Data = pattern.ReplaceAllString(row.Data, replace)
			row.Size = len(row.Data)
			editorAddInsertUndo(0, cy, row.Data)
			editorUpdateRow(row, Global.CurrentB)
		}
	}
	if matches > 0 {
		Global.Input = fmt.Sprintf("Replaced %d occurences on %d lines",
			matches, lines)
	} else {
		Global.Input = "No matches found"
	}
}

func gotoLine() {
	line, err := strconv.Atoi(editorPrompt("Go to line", nil))
	if err != nil {
		Global.Input = "Cancelled."
		return
	}
	line--
	if line < 0 {
		line = 0
	} else if line > Global.CurrentB.NumRows {
		line = Global.CurrentB.NumRows
	}
	Global.CurrentB.cy = line
	Global.Input = "Jumping to line " + strconv.Itoa(line+1)
}

func gotoChar() {
	line, err := strconv.Atoi(editorPrompt("Go to char", nil))
	if err != nil {
		Global.Input = "Cancelled."
		return
	}
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		return
	}
	datalen := len(Global.CurrentB.Rows[Global.CurrentB.cy].Data)
	if line < 0 {
		line = 0
	} else if line >= datalen {
		line = datalen
	}
	Global.CurrentB.cx = line
	Global.CurrentB.prefcx = Global.CurrentB.cx
	Global.Input = "Jumping to char " + strconv.Itoa(line)
}

func getOffsetInBuffer(buf *EditorBuffer) (int, int) {
	offset, total := 0, 0
	for i, row := range buf.Rows {
		total += row.Size + 1
		if i == buf.cy {
			offset += buf.cx
		} else if i < buf.cy {
			offset = total
		}
	}
	return offset, total - 1
}

func describeRune(ru rune) string {
	ri := int(ru)
	return fmt.Sprintf("%c (%d dec, %#o oct, %#02x hex)", ru, ri, ri, ri)
}

func whatCursorPosition() {
	cx, cy := Global.CurrentB.cx, Global.CurrentB.cy
	if cy >= Global.CurrentB.NumRows {
		Global.Input = "End of buffer"
		return
	}
	row := Global.CurrentB.Rows[cy]
	var ru rune
	if cx >= row.Size {
		ru = '\n'
	} else {
		ru, _ = utf8.DecodeRuneInString(row.Data[cx:])
	}
	offset, total := getOffsetInBuffer(Global.CurrentB)
	pc := 0
	if total > 0 {
		pc = (offset * 100) / (total)
	}
	Global.Input = fmt.Sprintf("Char: %s Offset: %d/%d (%d%%)", describeRune(ru),
		offset, total, pc)
}
