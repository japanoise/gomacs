package main

import (
	"strings"

	"github.com/zhemao/glisp/interpreter"
)

type EditorUndo struct {
	ins    bool
	region bool
	startl int
	endl   int
	startc int
	endc   int
	str    string
	prev   *EditorUndo
	paired bool
}

// Discards the last undo. Useful for the region functions, as they're made of
// regular insertion functions (which take care of their own undo)
func editorPopUndo() {
	old := Global.CurrentB.Undo
	if old == nil {
		return
	}
	Global.CurrentB.Undo = old.prev
}

func editorAddRegionUndo(ins bool, startc, endc, startl, endl int, str string) {
	old := Global.CurrentB.Undo
	ret := new(EditorUndo)
	ret.endl = endl
	ret.startl = startl
	ret.endc = endc
	ret.startc = startc
	ret.str = str
	ret.ins = ins
	ret.region = true

	if old == nil {
		ret.prev = nil
	} else {
		ret.prev = old
	}
	Global.CurrentB.Undo = ret
	Global.CurrentB.Redo = nil
}

func editorAddInsertUndo(startc, startl int, str string) {
	old := Global.CurrentB.Undo
	newlines := strings.Count(str, "\n")
	lastnl := 0
	if 0 < newlines {
		lastnl = strings.LastIndex(str, "\n") + 1
	}
	if old != nil && old != Global.CurrentB.SaveUndo &&
		old.ins && old.endc == startc && old.endl == startl {
		old.str += str
		old.endl += newlines
		if newlines <= 0 {
			old.endc += len(str)
		} else {
			old.endc = len(str[lastnl:])
		}
		old.region = old.region || newlines > 0
	} else {
		ret := new(EditorUndo)
		ret.startl = startl
		ret.startc = startc
		ret.str = str
		ret.ins = true
		ret.region = newlines > 0
		if newlines == 0 {
			ret.endl = startl
			ret.endc = startc + len(str)
		} else {
			ret.endl = startl + newlines
			ret.endc = len(str[lastnl:])
		}
		ret.prev = old
		Global.CurrentB.Undo = ret
	}
	Global.CurrentB.Redo = nil
}

func editorAddDeleteUndo(startc, endc, startl, endl int, str string) {
	old := Global.CurrentB.Undo
	ins := false
	app := false
	if old != nil {
		app = old.startl == startl && old.endl == endl && old.ins == ins && old != Global.CurrentB.SaveUndo
		if app {
			if ins {
				app = old.endc == startc
			} else {
				app = old.startc == endc
			}
		}
	}
	if app {
		Global.CurrentB.Redo = nil
		//append to group things together, ala gnu
		if ins {
			old.str += str
			old.endc = endc
		} else {
			old.str = str + old.str
			old.startc = startc
		}
	} else {
		ret := new(EditorUndo)
		ret.endl = endl
		ret.startl = startl
		ret.endc = endc
		ret.startc = startc
		ret.str = str
		ret.ins = ins
		ret.region = false

		if old == nil {
			ret.prev = nil
		} else {
			ret.prev = old
		}
		Global.CurrentB.Undo = ret
		Global.CurrentB.Redo = nil
	}
}

func editorDoUndo(tree *EditorUndo) bool {
	if tree == nil {
		return false
	}
	if tree.region {
		if tree.ins {
			bufKillRegion(Global.CurrentB, tree.startc, tree.endc, tree.startl, tree.endl)
		} else {
			spitRegion(tree.startc, tree.startl, tree.str)
		}
		return true
	}
	if tree.ins {
		// Insertion
		if tree.startl == tree.endl {
			// Basic string insertion
			editorRowDelChar(Global.CurrentB.Rows[tree.startl], Global.CurrentB,
				tree.startc, len(tree.str))
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.startl
			return true
		} else if tree.startl == -1 {
			// inserting a string on the last line
			editorDelRow(Global.CurrentB.NumRows - 1)
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.endl
			return true
		} else {
			// inserting a line
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.startl
			editorRowAppendStr(Global.CurrentB.Rows[tree.startl], Global.CurrentB, tree.str)
			editorDelRow(tree.endl)
			return true
		}
	} else {
		// Deletion
		if tree.startl == tree.endl {
			// Character or word deletion
			editorRowInsertStr(Global.CurrentB.Rows[tree.startl], Global.CurrentB,
				tree.startc, tree.str)
			Global.CurrentB.cx = tree.endc
			Global.CurrentB.cy = tree.startl
			return true
		} else {
			// deleting a line
			editorInsertRow(tree.startl, Global.CurrentB.Rows[tree.startl].Data[:tree.endc])
			row := Global.CurrentB.Rows[tree.endl]
			row.Data = tree.str
			row.Size = len(row.Data)
			Global.CurrentB.Rows[tree.startl].Size = len(Global.CurrentB.Rows[tree.startl].Data)
			editorUpdateRow(row, Global.CurrentB)
			editorUpdateRow(Global.CurrentB.Rows[tree.startl], Global.CurrentB)
			return true
		}
	}
}

func editorDoRedo(tree *EditorUndo) {
	if tree.region {
		if tree.ins {
			spitRegion(tree.startc, tree.startl, tree.str)
		} else {
			bufKillRegion(Global.CurrentB, tree.startc, tree.endc, tree.startl, tree.endl)
		}
		return
	}
	if tree.ins {
		if tree.startl == tree.endl {
			spitRegion(tree.startc, tree.startl, tree.str)
		} else if tree.startl == -1 {
			editorAppendRow(tree.str)
			Global.CurrentB.cx = tree.endc
			Global.CurrentB.cy = Global.CurrentB.NumRows - 1
		} else {
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.startl
			editorInsertNewline(false)
			editorPopUndo()
		}
	} else {
		if tree.startl == tree.endl {
			bufKillRegion(Global.CurrentB, tree.startc, tree.endc, tree.startl, tree.endl)
		} else {
			Global.CurrentB.cy = tree.endl
			Global.CurrentB.cx = 0
			editorDelChar()
			editorPopUndo()
		}
	}
}

func editorUndoAction() {
	r := Global.CurrentB.Redo
	succ := editorDoUndo(Global.CurrentB.Undo)
	paired := Global.CurrentB.Undo != nil && Global.CurrentB.Undo.paired
	if succ {
		Global.CurrentB.Redo = Global.CurrentB.Undo
		Global.CurrentB.Undo = Global.CurrentB.Undo.prev
		Global.CurrentB.Redo.prev = r
		if paired {
			editorUndoAction()
		}
	} else {
		Global.Input = "No further undo information."
	}
	if Global.CurrentB.Undo == Global.CurrentB.SaveUndo {
		Global.CurrentB.Dirty = false
	}
}

func doOneRedo(env *glisp.Glisp) {
	if Global.CurrentB.Redo == nil {
		Global.Input = "No further redo information."
	} else {
		r := Global.CurrentB.Redo
		editorDoRedo(Global.CurrentB.Redo)
		Global.CurrentB.Redo = r.prev
		r.prev = Global.CurrentB.Undo
		Global.CurrentB.Undo = r
		if r == Global.CurrentB.SaveUndo {
			Global.CurrentB.Dirty = false
		}
		if Global.CurrentB.Redo != nil && Global.CurrentB.Redo.paired {
			doOneRedo(env)
		}
	}
}

func editorRedoAction(env *glisp.Glisp) {
	micromode("C-_", "Press C-_ or C-/ to redo again", env, doOneRedo)
}
