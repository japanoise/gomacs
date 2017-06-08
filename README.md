# Gomacs - Go Powered Emacs!

![A screenshot showing Gomacs editing this README and its own source code](gomacs.png)

Gomacs is an Emacs clone for the terminal. Unlike many other mini-Emacsen, it
has an embedded Lisp, [powered by Glisp.](https://github.com/zhemao/glisp)
This puts it in the realm of true Emacs! It also supports primitive syntax
highlighting for C and Go.

Closely follows
[the modified version of Kilo found in this tutorial](http://viewsourcecode.org/snaptoken/kilo)
in terms of the inner workings of the editor. Inspiration also comes from
[the suckless editor Sandy.](http://tools.suckless.org/sandy)

## Installation

If your $GOPATH (or %GOPATH% under Windows) is set correctly, you should be able
to run:

    go get github.com/japanoise/gomacs

## Usage

    gomacs [options] file

Currently, the only supported option is -s, which disables syntax highlighting.

## Keybindings

Gomacs uses the standard Emacs keybindings. Of course, not all are implemented
yet - I'll try to keep this list up to date!

### Basics

- `C-g` - Cancel an action/keybinding/whatever
- `C-x C-c` - Save buffers and quit
- `C-x C-s` - Save buffer
- `C-_` - Undo (`C-/` also works)
- `C-x C-_` - Redo (`C-x C-/` also works) - press `C-_` or `C-/` again to redo
  more actions
- `C-z` - Suspend Gomacs (Linux only)
- `M-x` - Run named command
- `<f12>` - Panic key - quit emacs immediately without saving changes. Useful if
  Glisp falls down (which may happen if you do a lot of hacking on the editor's
  internals)

### Getting help

- `<f1>` - Quickhelp
- `C-h b` - Show key bindings
- `C-h c` - Describe keybinding briefly
- `C-h m` - Show modes active in buffer

### View operations

- `C-x C-f` - find file
- `C-x b` - switch buffer
- `C-x k` - kill buffer
- `C-x 2` - open a new window
- `C-x o` - switch to other window
- `C-x 0` - delete selected window
- `C-x 1` - maximise selected window (deleting the others)
- `C-x 4 C-f` - find file in other window (creating one if there's only one window)
- `C-x 4 b` - switch buffer in other window
- `C-l` - Centre view on current line
- `M-g g` or `M-g M-g` - Go to line (prompt)
- `M-g c` - Go to char (prompt)

### Cursor Movement

- `C-f` or `RIGHT` - Move cursor forward one character
- `M-f` - Move cursor forward one word
- `C-b` or `LEFT` - Move cursor backward one character
- `M-b` - Move cursor backward one word
- `C-p` or `UP` - Move cursor to previous line
- `C-n` or `DOWN` - Move cursor to next line
- `C-v` or `next` (Page Down) - Move cursor forward a screen
- `M-v` or `prior` (Page Up) - Move cursor backward a screen
- `C-s` - Incremental search
- `M-<` - Go to start of buffer
- `M->` - Go to end of buffer

### Deletion and Transposition

- `M-l` - Lowercase forward word
- `M-u` - Uppercase forward word
- `C-d` or `deletechar` - Delete forwards
- `backspace` - Delete backwards
- `M-d` - Delete forward word
- `M-<backspace>` or `M-D` (Meta-Shift-D) - delete backward word (`M-<deletechar>`
  does not work due to a fault either in Termbox or my terminal. If it works in
  your terminal, feel free to bind it.)
- `C-k` - Delete to end of line

### Region operations

- `C-@` - Set Mark (`C-<space>` also works)
- `C-w` - Kill (cut) region between mark and cursor
- `M-w` - Copy region between mark and cursor
- `C-y` - Yank (paste) previously copied or killed region
- `C-x C-u` - Uppercase region
- `C-x C-l` - Lowercase region

### Misc

- `C-x (` - Start recording a macro
- `C-x )` - Stop recording a macro
- `C-x e` - Stop recording a macro and execute it (repeat by pressing `e`)

## Customization

Emacs loads from ~/.gomacs.lisp on startup and executes the content of this file.
Check out the Glisp documentation for information on how the language works!
Some functions to get you started…

- `(emacsbindkey arg1 arg2..)` - Bind arg1 (in standard Emacs C-\*/M-\* notation,
  subsequent keypresses space seperated) to the Lisp function or named command
  arg2. arg1 must be a string; arg2 can be a function or a string. If arg2 is a
  function, any additional args will be used as its arguments when run.
- `(emacsdefinecmd arg1 arg2..)` - Define a command with name arg1 that runs the
  function arg2 with any aditional arguments as its arguments. arg1 must be a
  string; arg2 must be a function.
- `(setsofttab arg)` - Enable (true) or disable (false) the use of soft tabs
  (spaces for indentation). arg must be a boolean.
- `(settabstop arg)` - Set the width of \t characters in cells. If using soft
  tabs, this also sets the number of spaces that will be inserted when you press
  the Tab key. arg must be an integer.
- `(gettabstr)` - returns what the Tab key inserts, either "\t" or some number
  of spaces.
- `(disablesyntax arg)` - Enable (false) or disable (true) syntax highlighting.
  arg must be a boolean.

## Modes

Each buffer has a number of modes activated. When a new buffer is opened, the
modes are copied from the "default" set. Modes are added to this set with the
lisp function `(adddefaultmode <mode>)` and removed with the function
`(remdefaultmode <mode>)`.
They can be toggled with the command `toggle-mode`, and you can view the current
buffer's activated modes with the `show-modes` command. Here's what each mode
does:

- `terminal-title-mode` - use an escape sequence to set the terminal title.
- `line-number-mode` - display line numbers on the left edge of the buffer.
- `auto-indent-mode` - copy indentation from previous line when inserting a
  newline.

## Why?

I wanted an emacs to run in my terminal when Real Emacs wasn't an option.
Other terminal based Emacsen are available, but they each have their problems:

- mg doesn't do unicode.
- uemacs is effectively unmaintained (except for a patch from Torvalds once
  every few years) and lacks undo.
- godit is good, but not hackable. Furthermore, while I appreciate the
  "religious" approach in principle, in practice I like being able to customize
  my editor.
- sandy is too suckless-y. They build good components, but their tools often
  lack completeness. Plus it doesn't *really* use Emacs bindings, but
  emacs-like bindings, ala bash or zsh's "emacs mode". It's also unmaintained.

Furthermore, apart from sandy, none of these editors support syntax highlighting.

Finally, and most importantly, it's a fun project I've always wanted to do ;)

## Credits

Thanks to nsf for termbox and godit (from which the suspend code comes.)
