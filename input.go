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
