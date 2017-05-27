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
	return termutil.PromptWithCallback(prompt, func(int, int) { editorRefreshScreen() }, callback)
}

func editorChoiceIndex(title string, choices []string, def int) int {
	return termutil.ChoiceIndex(title, choices, def)
}

func ParseTermboxEvent(ev termbox.Event) string {
	return termutil.ParseTermboxEvent(ev)
}

func editorYesNoPrompt(p string, allowcancel bool) (bool, error) {
	if allowcancel {
		return termutil.YesNo(p, func(int, int) { editorRefreshScreen() }), nil
	} else {
		r, err := termutil.YesNoCancel(p, func(int, int) { editorRefreshScreen() })
		if err != nil {
			Global.Input = "Cancelled."
		}
		return r, err
	}
}
