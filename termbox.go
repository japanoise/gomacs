package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
)

func ParseTermboxEvent(ev termbox.Event) string {
	if ev.Ch == 0 {
		switch ev.Key {
		case termbox.KeyBackspace:
		case termbox.KeyBackspace2:
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
