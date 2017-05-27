package main

func getCurrentWindow() int {
	for i, win := range Global.Windows {
		if win == Global.CurrentB {
			return i
		}
	}
	return -1
}

func splitWindows() {
	Global.Windows = append(Global.Windows, Global.CurrentB)
}

func closeOtherWindows() {
	Global.Windows = []*EditorBuffer{Global.CurrentB}
}

func closeThisWindow() {
	if len(Global.Windows) == 1 {
		Global.Input = "Can't delete the only window!"
		return
	}
	i := getCurrentWindow()
	copy(Global.Windows[i:], Global.Windows[i+1:])
	Global.Windows[len(Global.Windows)-1] = nil
	Global.Windows = Global.Windows[:len(Global.Windows)-1]
	if i >= len(Global.Windows) {
		Global.CurrentB = Global.Windows[len(Global.Windows)-1]
	} else {
		Global.CurrentB = Global.Windows[i]
	}
}

func switchWindow() {
	cur := getCurrentWindow()
	cur++
	if cur >= len(Global.Windows) {
		Global.CurrentB = Global.Windows[0]
	} else {
		Global.CurrentB = Global.Windows[cur]
	}
}

func editorFindFile() {
	fn := editorPrompt("Find File", nil)
	if fn == "" {
		return
	}
	buffer := &EditorBuffer{}
	Global.Buffers = append(Global.Buffers, buffer)
	i := getCurrentWindow()
	if i < 0 {
		Global.Windows = []*EditorBuffer{buffer}
	} else {
		Global.Windows[i] = buffer
	}
	Global.CurrentB = buffer
	EditorOpen(fn)
}

func (e *EditorBuffer) getFilename() string {
	if e.Filename == "" {
		return "*unnamed buffer*"
	}
	return e.Filename
}

func bufferChoiceList() ([]string, int) {
	choices := []string{}
	def := 0
	for i, buf := range Global.Buffers {
		if buf == Global.CurrentB {
			def = i
		}
		d := ""
		if buf.Dirty {
			d = "[M] "
		}
		choices = append(choices, d+buf.getFilename())
	}
	return choices, def
}

func editorSwitchBuffer() {
	choices, def := bufferChoiceList()
	in := editorChoiceIndex("Switch buffer", choices, def)
	i := getCurrentWindow()
	Global.Windows[i] = Global.Buffers[in]
	Global.CurrentB = Global.Buffers[in]
}

func killBuffer() {
	// Deleting the last buffer will result in painful death!
	if len(Global.Buffers) == 1 {
		Global.Input = "Can't kill the last buffer!"
		return
	}

	// Choose a buffer to kill, and leave if the user cancels.
	choices, _ := bufferChoiceList()
	i := editorChoiceIndex("Kill buffer", choices, -1)
	if i == -1 {
		Global.Input = "Cancel."
		return
	}
	kb := Global.Buffers[i]

	// Prompt the user if buffer modified
	if kb.Dirty {
		c, _ := editorYesNoPrompt("Buffer has unsaved changes; kill anyway?", false)
		if !c {
			return
		}
	}

	// Find a replacement buffer.
	// If the killed buffer is selected, select replacement.
	var rb *EditorBuffer
	for _, buf := range Global.Buffers {
		if buf != kb {
			rb = buf
			break
		}
	}
	if Global.CurrentB == kb {
		Global.CurrentB = rb
	}

	// Delete the killed buffer.
	copy(Global.Buffers[i:], Global.Buffers[i+1:])
	Global.Buffers[len(Global.Buffers)-1] = nil
	Global.Buffers = Global.Buffers[:len(Global.Buffers)-1]

	// Replace any instance of the killed buffer in the window list with replacement
	for wi, win := range Global.Windows {
		if win == kb {
			Global.Windows[wi] = rb
		}
	}
}

func callFunOtherWindow(f func()) {
	if len(Global.Windows) == 1 {
		splitWindows()
	}
	switchWindow()
	f()
}
