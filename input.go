package main

import (
	"github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
)

func InitTerm() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt)
}

func editorGetKey() string {
	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
		} else if ev.Type == termbox.EventKey {
			return ParseTermboxEvent(ev)
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
				Global.Prompt = "Multiple choices"
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
				Global.Prompt = ""
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

func InsertRaw() {
	if Global.CurrentB.hasMode("no-self-insert-mode") {
		Global.Input = "Can't insert right now"
		return
	}
	done := false
	chara := ""
	for !done {
		data := make([]byte, 4)
		termbox.PollRawEvent(data)
		parsed := termbox.ParseEvent(data)
		if parsed.Type == termbox.EventKey {
			if data[3] == 0 {
				if data[2] == 0 {
					if data[1] == 0 {
						chara = string(data[:1])
					} else {
						chara = string(data[:2])
					}
				} else {
					chara = string(data[:3])
				}
			} else {
				chara = string(data)
			}
			done = true
		} else if parsed.Type == termbox.EventResize {
			editorRefreshScreen()
		}
	}
	editorInsertStr(chara)
}
