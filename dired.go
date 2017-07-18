package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/zhemao/glisp/interpreter"
)

func DiredMode(env *glisp.Glisp) {
	dir, perr := filepath.Abs("./")
	if perr != nil {
		Global.Input = perr.Error()
		AddErrorMessage(perr.Error())
		return
	}
	done := false
	for !done {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			Global.Input = err.Error()
			AddErrorMessage(err.Error())
			return
		}
		choices := []string{"<Cancel>", "../"}
		for _, f := range files {
			suffix := ""
			if f.IsDir() {
				suffix += "/"
			}
			suffix += " " + f.Mode().String()
			choices = append(choices, f.Name()+suffix)
		}
		choice := editorChoiceIndex("Dired: "+dir, choices, 0)
		filechosen := choice - 2
		if filechosen == -2 {
			return
		} else if filechosen == -1 {
			dir += "/.."
		} else if files[filechosen].IsDir() {
			dir += "/" + files[filechosen].Name()
		} else {
			openFile(dir+"/"+files[filechosen].Name(), env)
			done = true
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			Global.Input = err.Error()
			AddErrorMessage(err.Error())
			return
		}
	}
}
