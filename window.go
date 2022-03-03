package main

import (
	termbox "github.com/nsf/termbox-go"
	glisp "github.com/glycerine/zygomys/zygo"
)

type winTree struct {
	split   bool
	hor     bool
	focused bool
	buf     *EditorBuffer
	childLT *winTree
	childRB *winTree
	Parent  *winTree
}

func getFocusWindow() *winTree {
	return getWindowWithProps(func(t *winTree) bool { return t.focused },
		Global.WindowTree)
}

func getWindowWithProps(f func(*winTree) bool, t *winTree) *winTree {
	if !t.split {
		if f(t) {
			return t
		}
		return nil
	}
	ret := getWindowWithProps(f, t.childLT)
	if ret == nil {
		return getWindowWithProps(f, t.childRB)
	}
	return ret

}

func (t *winTree) mapTree(f func(*winTree)) {
	if t.split {
		t.childLT.mapTree(f)
		t.childRB.mapTree(f)
	} else {
		f(t)
	}
}

func vSplit() {
	win := getFocusWindow()
	win.focused = false
	win.split = true
	win.hor = false
	win.childLT = &winTree{false, false, true, win.buf, nil, nil, win}
	win.childRB = &winTree{false, false, false, win.buf, nil, nil, win}
	win.buf = nil
}

func hSplit() {
	win := getFocusWindow()
	win.focused = false
	win.split = true
	win.hor = true
	win.childLT = &winTree{false, false, true, win.buf, nil, nil, win}
	win.childRB = &winTree{false, false, false, win.buf, nil, nil, win}
	win.buf = nil
}

func closeOtherWindows() {
	win := getFocusWindow()
	Global.WindowTree = win
	Global.WindowTree.Parent = nil
}

func closeThisWindow() {
	win := getFocusWindow()

	if win == Global.WindowTree {
		Global.Input = "Can't delete the only window!"
		return
	}

	parent := win.Parent

	if parent.childLT == win {
		parent.buf = parent.childRB.buf
	} else {
		parent.buf = parent.childLT.buf
	}
	parent.childLT = nil
	parent.childRB = nil
	parent.split = false
	parent.focused = true
}

func switchWindow() {
	win := getFocusWindow()

	if win == Global.WindowTree {
		Global.Input = "Only window"
		return
	}

	win.focused = false

	parent := win.Parent
	cwin := win

	for parent != nil {
		if parent.childLT == cwin {
			parent.childRB.setFocus()
			return
		}
		cwin = parent
		parent = cwin.Parent
	}

	Global.WindowTree.setFocus()
}

func switchWindowOrientation() {
	win := getFocusWindow()

	if win == Global.WindowTree {
		Global.Input = "Only window"
		return
	}

	parent := win.Parent
	parent.hor = !parent.hor
}

func swapWindows() {
	win := getFocusWindow()

	if win == Global.WindowTree {
		Global.Input = "Only window"
		return
	}

	parent := win.Parent
	bak := parent.childLT
	parent.childLT = parent.childRB
	parent.childRB = bak
}

func (t *winTree) setFocus() {
	if t.split {
		t.childLT.setFocus()
	} else {
		t.focused = true
		Global.CurrentB = t.buf
	}
}

func (t *winTree) draw(x, y, wx, wy int) {
	if t.split {
		if t.hor {
			t.childLT.draw(x, y, wx/2, wy)

			for i := 0; i < wy; i++ {
				termbox.SetCell(
					x+(wx/2), y+i, '│',
					termbox.ColorDefault,
					termbox.ColorDefault)
			}

			termbox.SetCell(x+wx/2, y+wy, ' ',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)

			t.childRB.draw(x+(wx/2)+1, y, (wx / 2), wy)
		} else {
			t.childLT.draw(x, y, wx, wy/2)
			t.childRB.draw(x, y+1+(wy/2), wx, (wy/2)-1)
		}
		return
	}
	gutter := 0
	if t.buf.hasMode("line-number-mode") && t.buf.NumRows > 0 {
		gutter = GetGutterWidth(t.buf.NumRows)
	}

	editorDrawStatusLine(x, y+wy, wx, t)
	editorScroll(wx-gutter, wy)

	if t.buf.regionActive {
		t.buf.recalcRegion()
	}

	if t.focused {
		Global.CurrentBHeight = wy
		editorDrawRowsFocused(x, y, x+wx, y+wy, t.buf, gutter)
	} else {
		editorDrawRows(x, y, x+wx, y+wy, t.buf, gutter)
	}
}

func editorWriteFile(env *glisp.Zlisp) {
	fn := tabCompletedEditorPrompt("Write File", tabCompleteFilename)
	if fn == "" {
		return
	}
	Global.CurrentB.Filename = fn
	Global.CurrentB.UpdateRenderName()
	EditorSave(env)
}

func editorVisitFile(env *glisp.Zlisp) {
	fn := tabCompletedEditorPrompt("Visit File", tabCompleteFilename)
	if fn == "" {
		return
	}
	oldcb := Global.CurrentB
	openFile(fn, env)
	Global.WindowTree.mapTree(func(t *winTree) {
		if t.buf == oldcb {
			t.buf = Global.CurrentB
		}
	})
	for i, buf := range Global.Buffers {
		if buf == oldcb {
			killGivenBuffer(i)
			return
		}
	}
}

func editorFindFile(env *glisp.Zlisp) {
	fn := tabCompletedEditorPrompt("Find File", tabCompleteFilename)
	if fn == "" {
		return
	}
	openFile(fn, env)
}

func openFile(fn string, env *glisp.Zlisp) {
	buffer := &EditorBuffer{}
	Global.Buffers = append(Global.Buffers, buffer)

	win := getFocusWindow()
	if win == nil {
		Global.WindowTree = &winTree{}
		Global.WindowTree.focused = true
		Global.WindowTree.buf = buffer
	} else {
		win.buf = buffer
	}

	Global.CurrentB = buffer
	ferr := EditorOpen(fn, env)
	if ferr != nil {
		Global.Input = ferr.Error()
		AddErrorMessage(ferr.Error())
		Global.CurrentB.Rows = make([]*EditorRow, 1)
		Global.CurrentB.Rows[0] = &EditorRow{Global.CurrentB.NumRows,
			0, "", 0, "", nil, nil, 0}
	}
}

func (e *EditorBuffer) getFilename() string {
	if e.Filename == "" {
		return "*unnamed buffer*"
	}
	return e.Filename
}

func (e *EditorBuffer) getRenderName() string {
	if e.Filename == "" {
		return "*unnamed buffer*"
	}
	return e.Rendername
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
	in := editorChoiceIndex("Switch buffer", append([]string{"View Messages"}, choices...), def+1)
	if in == 0 {
		showMessages(Global.messages...)
	} else {
		win := getFocusWindow()
		win.buf = Global.Buffers[in-1]
		Global.CurrentB = Global.Buffers[in-1]
	}
}

func killGivenBuffer(i int) {
	// Deleting the last buffer will result in painful death!
	if len(Global.Buffers) == 1 {
		Global.Input = "Can't kill the last buffer!"
		return
	}

	if i <= -1 || i >= len(Global.Buffers) {
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
	Global.WindowTree.mapTree(func(t *winTree) {
		if t.buf == kb {
			t.buf = rb
		}
	})

	// Delete any mentions of this buffer in the registers
	for _, reg := range Global.Registers.Registers {
		if reg.Type == RegisterPos && reg.PosBuffer == kb {
			reg.Type = RegisterInvalid
			reg.PosBuffer = nil
		}
	}
}

func killBuffer() {
	// Deleting the last buffer will result in painful death!
	if len(Global.Buffers) == 1 {
		Global.Input = "Can't kill the last buffer!"
		return
	}

	// Choose a buffer to kill
	choices, _ := bufferChoiceList()
	killGivenBuffer(editorChoiceIndex("Kill buffer", choices, -1))
}

func callFunOtherWindow(f func()) {
	if !Global.WindowTree.split {
		vSplit()
	}
	switchWindow()
	f()
}

func callFunOtherWindowAndGoBack(f func()) {
	oldfw := getFocusWindow()
	callFunOtherWindow(f)
	getFocusWindow().focused = false
	oldfw.setFocus()
}

func getIndexOfCurrentBuffer() int {
	win := getFocusWindow()
	for i, buf := range Global.Buffers {
		if buf == win.buf {
			return i
		}
	}
	return -1
}

func KillBufferAndWindow() {
	bufi := getIndexOfCurrentBuffer()
	closeThisWindow()
	killGivenBuffer(bufi)
}
