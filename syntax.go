package main

import (
	"github.com/nsf/termbox-go"
	"strings"
)

const (
	HL_HI_NUMBERS int = 1 << iota
	HL_HI_STRINGS
)

type EditorSyntax struct {
	filetype                string
	filematch               []string
	flags                   int
	single_line_comment     string
	multiline_comment_start string
	multiline_comment_end   string
	keywords                []string
}

var HLDB []EditorSyntax = []EditorSyntax{
	EditorSyntax{
		"c",
		[]string{".c", ".h", ".cpp"},
		HL_HI_NUMBERS | HL_HI_STRINGS,
		"//",
		"/*",
		"*/",
		[]string{"switch", "if", "while", "for", "break", "continue", "return", "else",
			"struct", "union", "typedef", "static", "enum", "class", "case",
			"int|", "long|", "double|", "float|", "char|", "unsigned|", "signed|",
			"void|", ""},
	},
	EditorSyntax{
		"golang",
		[]string{".go"},
		HL_HI_NUMBERS | HL_HI_STRINGS,
		"//",
		"/*",
		"*/",
		[]string{"append", "break", "cap", "case", "chan", "close", "complex", "const",
			"continue", "copy", "default", "defer", "else", "fallthrough", "for", "func",
			"go", "goto", "if", "iota", "imag", "import", "interface", "len", "make",
			"map", "new", "package", "panic", "print", "println", "range", "real",
			"recover", "return", "select", "struct", "switch", "type", "var",
			"bool|", "byte|", "complex128|", "complex64|", "error|", "float32|", "float64|", "int|",
			"int16|", "int32|", "int64|", "int8|", "uint|", "uint16|", "uint32|",
			"uint64|", "uint8|", "uintptr|", "string|", "rune|", "error|", "ComplexType|",
			"FloatType|", "IntegerType|", "Type|", "Type1|", ""},
	},
}

type EmacsColor byte

const (
	HlDefault EmacsColor = iota
	HlNumber
	HlSearch
	HlString
	HlEscape
	HlComment
	HlKeyword1
	HlKeyword2
	HlMlComment
)

func isdigit(rv rune) bool {
	return rv >= '0' && rv <= '9'
}

func isseperator(rv rune) bool {
	return rv == ' ' || rv == 0 || strings.ContainsRune(",.()+-/*=~%<>[];", rv)
}

// Welcome to hell. This function has the lovely job of highlighting a line.
// This is, of course, a monstrous task. Sorry.
func editorUpdateSyntax(row *EditorRow) {
	row.Hl = make([]EmacsColor, len(row.Render))
	if Global.CurrentB.Syntax == nil || Global.NoSyntax {
		return
	}

	syn := Global.CurrentB.Syntax

	keywords := syn.keywords
	scs := syn.single_line_comment
	mcs := syn.multiline_comment_start
	mce := syn.multiline_comment_end

	prev_sep := true
	in_string := false
	in_comment := (row.idx > 0 && Global.CurrentB.Rows[row.idx-1].hl_open_comment)
	var stringc rune
	skip := 0
	prev_hl := HlDefault

	for i, ru := range row.Render {
		if skip > 0 {
			skip--
			continue
		}
		if len(scs) > 0 && !in_string && !in_comment {
			if strings.HasPrefix(row.Render[i:], scs) {
				for j := range row.Hl[i:] {
					row.Hl[i+j] = HlComment
				}
				break
			}
		}
		if len(mcs) > 0 && len(mce) > 0 && !in_string {
			if in_comment {
				row.Hl[i] = HlMlComment
				if strings.HasPrefix(row.Render[i:], mce) {
					for j := range row.Hl[i : i+len(mce)] {
						row.Hl[i+j] = HlMlComment
					}
					skip = len(mce) - 1
					in_comment = false
					prev_sep = true
					continue
				} else {
					continue
				}
			} else if strings.HasPrefix(row.Render[i:], mcs) {
				for j := range row.Hl[i : i+len(mcs)] {
					row.Hl[i+j] = HlMlComment
				}
				skip = len(mcs) - 1
				in_comment = true
				continue
			}
		}
		if syn.flags&HL_HI_NUMBERS != 0 {
			if (isdigit(ru) && (prev_sep || prev_hl == HlNumber)) || (ru == '.' && prev_hl == HlNumber) {
				row.Hl[i] = HlNumber
				prev_hl = HlNumber
				prev_sep = false
				continue
			}
		}
		if syn.flags&HL_HI_STRINGS != 0 {
			if in_string {
				row.Hl[i] = HlString
				prev_hl = HlString
				prev_sep = false
				if ru == '\\' && i+1 < len(row.Hl) {
					row.Hl[i+1] = HlString
					skip = 1
					continue
				}
				if ru == stringc && prev_hl != HlEscape {
					in_string = false
				}
				continue
			} else {
				if ru == '\'' || ru == '"' {
					in_string = true
					stringc = ru
					row.Hl[i] = HlString
					prev_hl = HlString
					continue
				}
			}
		}
		if prev_sep {
			var kw string
			for _, kw = range keywords {
				klen := len(kw)
				if kw == "" {
					break
				}
				kw2 := kw[klen-1] == '|'
				if kw2 {
					kw = kw[:klen-1]
					klen--
				}
				nextsep := false
				if i+klen >= len(row.Hl) {
					nextsep = true
				} else if isseperator(rune(row.Render[i+klen])) {
					nextsep = true
				}
				if nextsep && strings.HasPrefix(row.Render[i:], kw) {
					for j := range row.Hl[i : i+klen] {
						if kw2 {
							row.Hl[i+j] = HlKeyword2
						} else {
							row.Hl[i+j] = HlKeyword1
						}
					}
					skip = klen - 1
					break
				}
			}
			if kw != "" {
				prev_sep = false
				prev_hl = HlKeyword1
				continue
			}
		}
		prev_sep = isseperator(ru)
		prev_hl = HlDefault
	}
	changed := row.hl_open_comment != in_comment
	row.hl_open_comment = in_comment
	if changed && row.idx+1 < Global.CurrentB.NumRows {
		editorUpdateSyntax(Global.CurrentB.Rows[row.idx+1])
	}
}

func editorSyntaxToColor(hl EmacsColor) termbox.Attribute {
	switch hl {
	case HlNumber:
		return termbox.ColorRed
	case HlSearch:
		return termbox.ColorBlue
	case HlEscape:
		fallthrough
	case HlString:
		return termbox.ColorMagenta
	case HlComment:
		fallthrough
	case HlMlComment:
		return termbox.ColorCyan
	case HlKeyword1:
		return termbox.ColorYellow
	case HlKeyword2:
		return termbox.ColorGreen
	}
	return termbox.ColorDefault
}

func editorSelectSyntaxHighlight(buf *EditorBuffer) {
	if buf.Filename == "" {
		buf.Syntax = nil
		return
	}
	i := strings.LastIndex(buf.Filename, ".")
	if i <= 0 {
		buf.Syntax = nil
		return
	}
	bft := buf.Filename[i:]
	for _, syn := range HLDB {
		for _, ft := range syn.filematch {
			if ft == bft {
				buf.Syntax = &syn
				if buf.Rows != nil {
					for _, row := range buf.Rows {
						editorUpdateSyntax(row)
					}
				}
				return
			}
		}
	}
	buf.Syntax = nil
}
