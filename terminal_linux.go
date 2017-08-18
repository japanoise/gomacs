package main

import "fmt"

func terminalTitle(buf *EditorBuffer) {
	fmt.Printf("\033]0;%s - gomacs\a", buf.getRenderName())
}
