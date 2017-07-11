package main

import (
	"fmt"
	"io"
	"os/exec"
)

func shellCmd(com string, args []string) (string, error) {
	cmd := exec.Command(com, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func doShellCmd() {
	com := editorPrompt("Command to run", nil)
	arg := editorPrompt("Argument 1 (Blank, C-c or C-g for none)", nil)
	args := []string{}
	for arg != "" {
		args = append(args, arg)
		arg = editorPrompt(fmt.Sprintf("Argument %d (Blank, C-c or C-g for none)", 1+len(args)), nil)
	}
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
	return string(out), err
}

func doShellCmdRegion() {
	com := editorPrompt("Command to run", nil)
	arg := editorPrompt("Argument 1 (Blank, C-c or C-g for none)", nil)
	args := []string{}
	for arg != "" {
		args = append(args, arg)
		arg = editorPrompt(fmt.Sprintf("Argument %d (Blank, C-c or C-g for none)", 1+len(args)), nil)
	}
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
		regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) {
			result, err := shellCmdWithInput(getRegionText(buf, startc, endc, startl, endl), com, args)
			if err == nil {
				showMessages(result)
			} else {
				showMessages(err.Error() + "\n" + result)
			}
		})
	}
}
