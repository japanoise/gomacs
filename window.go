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

func editorSwitchBuffer() {
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
		if buf.Filename == "" {
			choices = append(choices, d+"*unnamed buffer*")
		} else {
			choices = append(choices, d+buf.Filename)
		}
	}
	in := editorChoiceIndex("Switch buffer", choices, def)
	i := getCurrentWindow()
	Global.Windows[i] = Global.Buffers[in]
	Global.CurrentB = Global.Buffers[in]
}

func callFunOtherWindow(f func()) {
	if len(Global.Windows) == 1 {
		splitWindows()
	}
	switchWindow()
	f()
}
