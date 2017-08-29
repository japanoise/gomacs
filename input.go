package main

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
)

func InitTerm() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt | termbox.InputMouse)
}

func editorGetKey() string {
	for {
		// More hacking; if we've been waiting for some time, refresh the screen.
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
				Global.CurrentB.updateHighlighting()
			case _ = <-done:
				// Don't refresh the screen.
			}
		}()
		ev := termbox.PollEvent()
		done <- true
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
		} else if ev.Type == termbox.EventKey {
			return ParseTermboxEvent(ev)
		} else if ev.Type == termbox.EventMouse {
			return ParseMouseEvent(ev)
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

func editorPrompt(prompt string, callback func(string, string)) string {
	ret := termutil.PromptWithCallback(prompt, func(int, int) { editorRefreshScreen() }, callback)
	Global.Input = ret
	return ret
}

func tabCompletedEditorPrompt(prompt string, getCandidates func(string) []string) string {
	ret := termutil.DynamicPromptWithCallback(prompt, func(int, int) { editorRefreshScreen() }, func(query, key string) string {
		if key == "TAB" || key == "C-i" {
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
					key := editorGetKey()
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
