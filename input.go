package main

import (
	"errors"
	"fmt"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"unicode/utf8"
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

func editorPrompt(prompt string, callback func(string, string)) string {
	buffer := ""
	buflen := 0
	cursor := 0
	editorSetPrompt(prompt)
	defer editorSetPrompt("")
	for {
		_, y := termbox.Size()
		Global.Input = buffer
		editorRefreshScreen()
		termbox.SetCursor(utf8.RuneCountInString(prompt)+3+cursor, y-1)
		termbox.Flush()
		key := editorGetKey()
		switch key {
		case "C-c":
			fallthrough
		case "C-g":
			if callback != nil {
				callback(buffer, key)
			}
			return ""
		case "RET":
			if callback != nil {
				callback(buffer, key)
			}
			return buffer
		case "DEL":
			if buflen > 0 {
				r, rs := utf8.DecodeLastRuneInString(buffer)
				buffer = buffer[0 : buflen-rs]
				buflen -= rs
				cursor -= runewidth.RuneWidth(r)
			}
		default:
			if utf8.RuneCountInString(key) == 1 {
				r, _ := utf8.DecodeLastRuneInString(buffer)
				buffer += key
				buflen += len(key)
				cursor += runewidth.RuneWidth(r)
			}
		}
		if callback != nil {
			callback(buffer, key)
		}
	}
}

func editorChoiceIndex(title string, choices []string, def int) int {
	selection := def
	// Will need these in a smarter version of this function
	//offset := 0
	//sx, sy := termbox.Size()
	for {
		termbox.HideCursor()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		printstring(title, 0, 0)
		for i, s := range choices {
			printstring(s, 3, i+1)
		}
		printstring(">", 1, selection+1)
		termbox.Flush()
		key := editorGetKey()
		switch key {
		case "C-c":
			fallthrough
		case "C-g":
			return def
		case "UP":
			if selection > 0 {
				selection--
			}
		case "DOWN":
			if selection < len(choices)-1 {
				selection++
			}
		case "RET":
			return selection
		}
	}
}

func ParseTermboxEvent(ev termbox.Event) string {
	if ev.Ch == 0 {
		switch ev.Key {
		case termbox.KeyBackspace:
		case termbox.KeyBackspace2:
			if ev.Mod == termbox.ModAlt {
				return fmt.Sprintf("M-DEL")
			}
			return "DEL"
		case termbox.KeyTab:
			return "TAB"
		case termbox.KeyEnter:
			return "RET"
		case termbox.KeyArrowDown:
			return "DOWN"
		case termbox.KeyArrowUp:
			return "UP"
		case termbox.KeyArrowLeft:
			return "LEFT"
		case termbox.KeyArrowRight:
			return "RIGHT"
		case termbox.KeyPgdn:
			return "next"
		case termbox.KeyPgup:
			return "prior"
		case termbox.KeyHome:
			return "Home"
		case termbox.KeyEnd:
			return "End"
		case termbox.KeyDelete:
			if ev.Mod == termbox.ModAlt {
				return fmt.Sprintf("M-deletechar")
			}
			return "deletechar"
		case termbox.KeyInsert:
			return "insert"
		case termbox.KeyCtrlUnderscore:
			return "C-_"
		case termbox.KeyCtrlSpace:
			return "C-@" // ikr, weird. but try: C-h c, C-SPC. it's C-@.
		case termbox.KeySpace:
			return " "
		}
		if ev.Key <= 0x1A {
			return fmt.Sprintf("C-%c", 96+ev.Key)
		} else if ev.Key <= termbox.KeyF1 && ev.Key >= termbox.KeyF12 {
			if ev.Mod == termbox.ModAlt {
				return fmt.Sprintf("M-f%d", 1+(termbox.KeyF1-ev.Key))
			} else {
				return fmt.Sprintf("f%d", 1+(termbox.KeyF1-ev.Key))
			}
		}
	} else if ev.Mod == termbox.ModAlt {
		return fmt.Sprintf("M-%c", ev.Ch)
	}
	return string(ev.Ch)
}

func editorYesNoPrompt(p string, allowcancel bool) (bool, error) {
	Global.Input = ""
	var plen int
	if allowcancel {
		pm := p + " (y/n/C-g)"
		plen = utf8.RuneCountInString(pm) + 3
		editorSetPrompt(pm)
	} else {
		pm := p + " (y/n)"
		plen = utf8.RuneCountInString(pm) + 3
		editorSetPrompt(pm)
	}
	editorRefreshScreen()
	_, y := termbox.Size()

	termbox.SetCursor(plen, y-1)
	termbox.Flush()
	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
			_, y = termbox.Size()
			termbox.SetCursor(plen, y-1)
			termbox.Flush()
		} else if ev.Type == termbox.EventKey {
			if ev.Key == termbox.KeyCtrlG && allowcancel {
				Global.Input = "Cancelled."
				editorSetPrompt("")
				return false, errors.New("User cancelled")
			} else if ev.Ch == 'y' {
				editorSetPrompt("")
				return true, nil
			} else if ev.Ch == 'n' {
				editorSetPrompt("")
				return false, nil
			}
		}
	}
}
