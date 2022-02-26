package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	termutil "github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
)

func InitTerm() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt | termbox.InputMouse)
}

func editorGetKey() (string, bool) {
	for {
		// More hacking; if we've been waiting for some time, refresh the screen.
		doReHl := false
		timeout := make(chan bool, 1)
		done := make(chan bool, 1)
		go func() {
			time.Sleep(time.Duration(TIMEOUT))
			timeout <- true
		}()
		go func() {
			select {
			case _ = <-timeout:
				editorRefreshScreen()
			case _ = <-done:
				// Don't refresh the screen.
			}
		}()
		ev := termbox.PollEvent()
		done <- true
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
		} else if ev.Type == termbox.EventKey {
			return ParseTermboxEvent(ev), doReHl
		} else if ev.Type == termbox.EventMouse {
			return ParseMouseEvent(ev), doReHl
		}
	}
}

func editorGetKeyNoRefresh() string {
	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventKey {
			return ParseTermboxEvent(ev)
		}
	}
}

func recalcBuffer(result string) (string, int, int, int) {
	rlen := len(result)
	return result, rlen, 0, 0
}

func backwordWordIndex(buffer string, bufpos int) int {
	r, rs := utf8.DecodeLastRuneInString(buffer[:bufpos])
	ret := bufpos - rs
	r, rs = utf8.DecodeLastRuneInString(buffer[:ret])
	for ret > 0 && termutil.WordCharacter(r) {
		ret -= rs
		r, rs = utf8.DecodeLastRuneInString(buffer[:ret])
	}
	return ret
}

func forwardWordIndex(buffer string, bufpos int) int {
	r, rs := utf8.DecodeRuneInString(buffer[bufpos:])
	ret := bufpos + rs
	r, rs = utf8.DecodeRuneInString(buffer[ret:])
	for ret < len(buffer) && termutil.WordCharacter(r) {
		ret += rs
		r, rs = utf8.DecodeRuneInString(buffer[ret:])
	}
	return ret
}

//As prompt, but calls a function after every keystroke.
func PromptWithCallback(prompt string, refresh func(int, int), callback func(string, string)) string {
	if callback == nil {
		return DynamicPromptWithCallback(prompt, refresh, nil)
	} else {
		return DynamicPromptWithCallback(prompt, refresh, func(a, b string) string {
			callback(a, b)
			return a
		})
	}
}

//As prompt, but calls a function after every keystroke that can modify the query.
func DynamicPromptWithCallback(prompt string, refresh func(int, int), callback func(string, string) string) string {
	return EditDynamicWithCallback("", prompt, refresh, callback)
}

// EditDynamicWithCallback takes a default value, prompt, refresh
// function, and callback. It allows the user to edit the default
// value. It returns what the user entered.
func EditDynamicWithCallback(defval, prompt string, refresh func(int, int), callback func(string, string) string) string {
	var buffer string
	var bufpos, cursor, offset int
	if defval == "" {
		buffer = ""
		bufpos = 0
		cursor = 0
		offset = 0
	} else {
		x, _ := termbox.Size()
		buffer = defval
		bufpos = len(buffer)
		if termutil.RunewidthStr(buffer) > x {
			cursor = x - 1
			offset = len(buffer) + 1 - x
		} else {
			offset = 0
			cursor = termutil.RunewidthStr(buffer)
		}
	}
	iw := termutil.RunewidthStr(prompt + ": ")
	for {
		buflen := len(buffer)
		x, y := termbox.Size()
		if refresh != nil {
			refresh(x, y)
		}
		termutil.ClearLine(x, y-1)
		for iw+cursor >= x {
			offset++
			cursor--
		}
		for iw+cursor < iw {
			offset--
			cursor++
		}
		t, _ := trimString(buffer, offset)
		termutil.Printstring(prompt+": "+t, 0, y-1)
		termbox.SetCursor(iw+cursor, y-1)
		termbox.Flush()
		ev := termbox.PollEvent()
		if ev.Type != termbox.EventKey {
			continue
		}
		key := ParseTermboxEvent(ev)
		switch key {
		case "LEFT", "C-b":
			if bufpos > 0 {
				r, rs := utf8.DecodeLastRuneInString(buffer[:bufpos])
				bufpos -= rs
				cursor -= termutil.Runewidth(r)
			}
		case "RIGHT", "C-f":
			if bufpos < buflen {
				r, rs := utf8.DecodeRuneInString(buffer[bufpos:])
				bufpos += rs
				cursor += termutil.Runewidth(r)
			}
		case "C-a":
			fallthrough
		case "Home":
			bufpos = 0
			cursor = 0
			offset = 0
		case "C-e":
			fallthrough
		case "End":
			bufpos = buflen
			if termutil.RunewidthStr(buffer) > x {
				cursor = x - 1
				offset = buflen + 1 - x
			} else {
				offset = 0
				cursor = termutil.RunewidthStr(buffer)
			}
		case "C-y":
			bp := bufpos
			buffer, buflen, bufpos, cursor = recalcBuffer(
				buffer[:bufpos] + Global.Clipboard + buffer[bufpos:])
			bufpos = bp + len(Global.Clipboard)
			cursor = termutil.RunewidthStr(buffer[:bufpos])
		case "C-c":
			fallthrough
		case "C-g":
			if callback != nil {
				result := callback(buffer, key)
				if result != buffer {
					offset = 0
					buffer, buflen, bufpos, cursor = recalcBuffer(result)
				}
			}
			return defval
		case "RET":
			if callback != nil {
				result := callback(buffer, key)
				if result != buffer {
					offset = 0
					buffer, buflen, bufpos, cursor = recalcBuffer(result)
				}
			}
			return buffer
		case "C-d":
			fallthrough
		case "deletechar":
			if bufpos < buflen {
				r, rs := utf8.DecodeRuneInString(buffer[bufpos:])
				bufpos += rs
				cursor += termutil.Runewidth(r)
			} else {
				if callback != nil {
					result := callback(buffer, key)
					if result != buffer {
						offset = 0
						buffer, buflen, bufpos, cursor = recalcBuffer(result)
					}
				}
				continue
			}
			fallthrough
		case "DEL", "C-h":
			if buflen > 0 {
				if bufpos == buflen {
					r, rs := utf8.DecodeLastRuneInString(buffer)
					buffer = buffer[0 : buflen-rs]
					bufpos -= rs
					cursor -= termutil.Runewidth(r)
				} else if bufpos > 0 {
					r, rs := utf8.DecodeLastRuneInString(buffer[:bufpos])
					buffer = buffer[:bufpos-rs] + buffer[bufpos:]
					bufpos -= rs
					cursor -= termutil.Runewidth(r)
				}
			}
		case "C-u":
			buffer = ""
			buflen = 0
			bufpos = 0
			cursor = 0
			offset = 0
		case "M-DEL":
			if buflen > 0 && bufpos > 0 {
				delto := backwordWordIndex(buffer, bufpos)
				buffer = buffer[:delto] + buffer[bufpos:]
				buflen = len(buffer)
				bufpos = delto
				cursor = termutil.RunewidthStr(buffer[:bufpos])
			}
		case "M-d":
			if buflen > 0 && bufpos < buflen {
				delto := forwardWordIndex(buffer, bufpos)
				buffer = buffer[:bufpos] + buffer[delto:]
				buflen = len(buffer)
			}
		case "M-b":
			if buflen > 0 && bufpos > 0 {
				bufpos = backwordWordIndex(buffer, bufpos)
				cursor = termutil.RunewidthStr(buffer[:bufpos])
			}
		case "M-f":
			if buflen > 0 && bufpos < buflen {
				bufpos = forwardWordIndex(buffer, bufpos)
				cursor = termutil.RunewidthStr(buffer[:bufpos])
			}
		case "C-i", "TAB":
			buffer = buffer[:bufpos] + "\t" + buffer[bufpos:]
			bufpos++
			cursor++
		default:
			if utf8.RuneCountInString(key) == 1 {
				r, _ := utf8.DecodeLastRuneInString(buffer)
				buffer = buffer[:bufpos] + key + buffer[bufpos:]
				bufpos += len(key)
				cursor += termutil.Runewidth(r)
			}
		}
		if callback != nil {
			result := callback(buffer, key)
			if result != buffer {
				offset = 0
				buffer, buflen, bufpos, cursor = recalcBuffer(result)
			}
		}
	}
}

func editorPrompt(prompt string, callback func(string, string)) string {
	ret := PromptWithCallback(prompt, func(int, int) { editorRefreshScreen() }, callback)
	Global.Input = ret
	return ret
}

func tabCompletedEditorPrompt(prompt string, getCandidates func(string) []string) string {
	ret := DynamicPromptWithCallback(prompt, func(int, int) { editorRefreshScreen() }, func(query, key string) string {
		if key == "TAB" || key == "C-i" {
			// HACK: Remove tabs from query
			query = strings.ReplaceAll(query, "\t", "")
			if getCandidates == nil {
				return query
			}
			candidates := getCandidates(query)
			if candidates == nil {
				return query
			} else if len(candidates) == 1 {
				return candidates[0]
			} else if 0 < len(candidates) {
				choice := 0
				undecided := true
				editorSetPrompt("Multiple choices")
				defer editorSetPrompt("")
				cachedinput := Global.Input
				for undecided {
					Global.Input = candidates[choice]
					editorRefreshScreen()
					key, _ := editorGetKey()
					if key == "TAB" || key == "C-i" || key == "RIGHT" || key == "C-f" {
						choice++
						if choice == len(candidates) {
							choice = 0
						}
					} else if key == "LEFT" || key == "C-b" {
						choice--
						if choice == -1 {
							choice = len(candidates) - 1
						}
					} else if key == "C-c" || key == "C-g" {
						return query
					} else {
						undecided = false
					}
				}
				Global.Input = cachedinput
				return candidates[choice]
			} else {
				return query
			}
		} else {
			return query
		}
	})
	return ret
}

func editorChoiceIndex(title string, choices []string, def int) int {
	return termutil.ChoiceIndex(title, choices, def)
}

func showMessages(mesgs ...string) {
	termbox.HideCursor()
	termutil.DisplayScreenMessage(mesgs...)
}

func ParseTermboxEvent(ev termbox.Event) string {
	return termutil.ParseTermboxEvent(ev)
}

func ParseMouseEvent(ev termbox.Event) string {
	var mousestr string
	switch ev.Key {
	case termbox.MouseLeft:
		mousestr = "mouse1"
	case termbox.MouseMiddle:
		mousestr = "mouse2"
	case termbox.MouseRight:
		mousestr = "mouse3"
	case termbox.MouseRelease:
		mousestr = "up-mouse"
	case termbox.MouseWheelUp:
		mousestr = "mouse4"
	case termbox.MouseWheelDown:
		mousestr = "mouse5"
	}
	return fmt.Sprintf("<%s %d %d>", mousestr, ev.MouseX, ev.MouseY)
}

func editorYesNoPrompt(p string, noallowcancel bool) (bool, error) {
	if noallowcancel {
		return termutil.YesNo(p, func(int, int) { editorRefreshScreen() }), nil
	} else {
		r, err := termutil.YesNoCancel(p, func(int, int) { editorRefreshScreen() })
		if err != nil {
			Global.Input = "Cancelled."
		}
		return r, err
	}
}

func editorPressKey(p string, keys ...string) string {
	return termutil.PressKey(p, func(int, int) { editorRefreshScreen() }, keys...)
}

func GetRawChar() string {
	return termutil.GetRawChar(func(int, int) {
		editorRefreshScreen()
	})
}

func InsertRaw() {
	if Global.CurrentB.hasMode("no-self-insert-mode") {
		Global.Input = "Can't insert right now"
		return
	}
	editorInsertStr(GetRawChar())
}

func zapToChar() {
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		Global.Input = "End of buffer"
		return
	}
	Global.Input = "Zap to char: "
	editorRefreshScreen()
	chars := GetRawChar()
	Global.Input += chars
	zapru, size := utf8.DecodeLastRuneInString(chars)
	for _, row := range Global.CurrentB.Rows[Global.CurrentB.cy:] {
		thisrow := row.idx == Global.CurrentB.cy
		for in, ru := range row.Data {
			if ru == zapru && !(thisrow && in < Global.CurrentB.cx) {
				Global.Clipboard = bufKillRegion(Global.CurrentB, Global.CurrentB.cx, in+size, Global.CurrentB.cy, row.idx)
				return
			}
		}
	}
	Global.Input = "Search failed: " + chars
}
