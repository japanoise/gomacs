package main

import (
	"io"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/zhemao/glisp/interpreter"
)

func shellCmd(com string, args []string) (string, error) {
	cmd := exec.Command(com, args...)
	out, err := cmd.CombinedOutput()
	// Gomacs doesn't like trailing newlines; strip 'em
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return string(out), err
}

func shellCmdAction(com string, args []string) {
	result, err := shellCmd(com, args)
	if err == nil {
		if Global.SetUniversal {
			spitRegion(Global.CurrentB.cx, Global.CurrentB.cy, result)
		} else {
			showMessages(result)
		}
	} else {
		if Global.SetUniversal {
			spitRegion(Global.CurrentB.cx, Global.CurrentB.cy, err.Error()+"\n"+result)
		} else {
			showMessages(err.Error() + "\n" + result)
		}
		AddErrorMessage(err.Error())
	}
}

func doShellCmd() {
	var words []string
	var err error
	for len(words) == 0 {
		words, err = shellquote.Split(editorPrompt("Shell command", nil))
		if err != nil {
			Global.Input = err.Error()
			return
		}
	}
	shellCmdAction(words[0], words[1:])
}

func shellCmdWithInput(input, com string, args []string) (string, error) {
	cmd := exec.Command(com, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()
	out, err := cmd.CombinedOutput()
	if out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return string(out), err
}

func shellCmdRegion(com string, args []string) {
	if Global.SetUniversal {
		transposeRegionCmd(func(input string) string {
			result, err := shellCmdWithInput(input, com, args)
			if err == nil {
				return result
			} else {
				return err.Error() + "\n" + result
			}
		})
	} else {
		regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) string {
			result, err := shellCmdWithInput(getRegionText(buf, startc, endc, startl, endl), com, args)
			if err == nil {
				showMessages(result)
			} else {
				showMessages(err.Error() + "\n" + result)
			}
			return ""
		})
	}
}

func doShellCmdRegion() {
	var words []string
	var err error
	for len(words) == 0 {
		words, err = shellquote.Split(editorPrompt("Shell command", nil))
		if err != nil {
			Global.Input = err.Error()
			return
		}
	}
	shellCmdRegion(words[0], words[1:])
}

func replaceBufferWithShellCommand(buf *EditorBuffer, com string, args []string, env *glisp.Glisp) {
	if buf.NumRows == 0 {
		return
	}
	output, err := shellCmdWithInput(getRegionText(buf, 0, buf.Rows[buf.NumRows-1].Size, 0, buf.NumRows-1), com, args)
	if err != nil {
		showMessages(err.Error(), output)
		return
	}
	lines := strings.Split(output, "\n")
	ll := len(lines)
	buf.Rows = make([]*EditorRow, ll)
	buf.NumRows = ll
	for i, line := range lines {
		newrow := &EditorRow{}
		newrow.Data = line
		newrow.idx = i
		newrow.Size = len(line)
		rowUpdateRender(newrow)
		buf.Rows[i] = newrow
	}
	if buf.Highlighter != nil {
		buf.Highlight()
	}
	buf.Undo = nil
	buf.Redo = nil
	if buf.cy >= buf.NumRows {
		buf.cy = buf.NumRows - 1
	}
	if buf.cx >= buf.Rows[buf.cy].Size {
		buf.cx = buf.Rows[buf.cy].Size
	}
	editorRowCxToRx(buf.Rows[buf.cy])
	editorBufSave(buf, env)
}
