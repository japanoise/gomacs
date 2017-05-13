# Gomacs - Go Powered Emacs!

![A screenshot showing Gomacs editing this README and its own source code](gomacs.png)

Gomacs is an Emacs clone for the terminal. Unlike many other mini-Emacsen, it
has an embedded Lisp, [powered by Zygomys.](https://github.com/glycerine/zygomys)
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

Gomacs uses the standard Emacs keybindings. Of course, not all are implemented
yet - I'll try to keep this list up to date!

- `C-s` - Incremental search
- `M-<` - Go to start of buffer
- `M->` - Go to end of buffer
- `C-l` - Centre view on current line
- `C-x C-s` - save changes
- `C-x C-c` - quit
- `C-x C-f` - find file
- `C-x b` - switch buffer
- `C-x k` - kill buffer
- `C-x 2` - open a new window
- `C-x o` - switch to other window
- `C-x 0` - delete selected window
- `C-x 1` - maximise selected window (deleting the others)
- `C-x 4 C-f` - find file in other window (creating one if there's only one window)
- `C-x 4 b` - switch buffer in other window
- `C-_` - Undo (`C-/` also works)
- `C-@` - Set Mark (`C-<space>` also works)
- `C-w` - Kill (cut) region between mark and cursor
- `M-w` - Copy region between mark and cursor
- `C-y` - Yank (paste) previously copied or killed region
- `M-d` - Delete forward word
- `M-<backspace>` or `M-D` (Meta-Shift-D) - delete backward word (`M-<deletechar>`
  does not work due to a fault either in Termbox or my terminal. If it works in
  your terminal, feel free to bind it.)
- `C-k` - Delete to end of line
- `<f12>` - Panic key - quit emacs immediately without saving changes. Useful if
  Zygomys falls down (which may happen if you do a lot of hacking on the editor's
  internals)

## Customization

Emacs loads from ~/.gomacs.lisp on startup and executes the content of this file.
Check out the Zygomys documentation for information on how the language works!
Some functions to get you started…

- `(emacsbindkey arg1 arg2)` - Bind arg1 (in standard Emacs C-*/M-* notation,
  subsequent keypresses space seperated) to the Lisp code in arg2. Both args
  must be strings.
- `(setsofttab arg)` - Enable (true) or disable (false) the use of soft tabs
  (spaces for indentation). arg must be a boolean.
- `(settabstop arg)` - Set the width of \t characters in cells. If using soft
  tabs, this also sets the number of spaces that will be inserted when you press
  the Tab key. arg must be an integer.
- `(gettabstr)` - returns what the Tab key inserts, either "\t" or some number
  of spaces.
- `(disablesyntax arg)` - Enable (false) or disable (true) syntax highlighting.
  arg must be a boolean.

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
