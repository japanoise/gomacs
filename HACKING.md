# Hacking Gomacs

## Files in Gomacs

- bindata.go - syntax highlighting data to be embedded into the executable.
  Leave this file alone! If you add a new syntax highlighting definition,
  though, you can run `go-bindata syntax_files/*.yaml`
- commands.go - code to do with registering and storing mappings between
  keypresses and lisp functions or commands.
- dired.go - barebones implementation of dired-mode
- input.go - input from the user. Translating a termbox key event into an emacs
  binding string.
- lisp.go - dealing with the lisp interpreter.
- macro.go - macro and micromode functionality
- main.go - big ball of tar! Most row editing, buffer actions, etc done here, as
  well as the main loop. An ongoing project is to extract code from here and into
  dedicated files.
- modes.go - dealing with modes
- mouse.go - mouse handling code
- nav.go - navigation code
- paragraph.go - paragraph-based commands
- rectangle.go - rectangle-based commands
- registers.go - commands that save, load, and run from registers
- region.go - functions and commands for acting upon the selected region.
- render.go - rendering and drawing functions
- shell.go - commands that use external programs
- suspend.go - placeholder for non-POSIX platforms (which don't have suspend
  functionality)
  * suspend_posix.go - suspend functionality for POSIX systems
- syntax.go - syntax highlighting functionality lives here.
- undo.go - creating, storing and destroying undo data. Doing undos and redos.
- window.go - window manipulation code.
- word.go - acting upon words.

## Gomacs' Parentage

Gomacs' very very primitive functions are based on [Kilo.](http://viewsourcecode.org/snaptoken/kilo)
However, there's an element of separation going on. The EditorState struct
stores the global state. It is referred to as Global in the code. Here the data
is subdivided further; we have the idea of "buffers," which are open files, and
"windows," which are files visible on screen. Both are stored using the EditorBuffer
struct. Global.CurrentB is the buffer we're currently editing - the window code
also uses this to determine which window is currently focused.

Another way Gomacs is not like Kilo is in the drawing code. We actually delegate
this to a library, [Termbox.](https://github.com/nsf/termbox-go) Termbox
provides us with some drawing primitives, but it mostly leaves us on our own.
Most of this functionality is in render.go. Some stuff is delegated to my
library [termutil;](https://github.com/japanoise/termbox-util) this can be
found in input.go, with a little in render.go and small parts scattered around
the other files.

Now, apart from these differences, a Kilo hacker will notice a lot of
similarities. Buffers are a wrapper around a list of EditorRows. Each of these
has a data field (the actual string from the file), a render string (what's
shown on screen, determined from the data) and a highlighting data array (which
helps the editor determine what colour to print strings in on screen).

Syntax highlighting is broadly similar to [Micro,](https://github.com/zyedidia/micro)
Gomacs' other main parent (we even both share some code from [Godit,](https://github.com/nsf/godit)
namely the suspend-on-Linux functionality). Syntax files live in the `syntax_files/`
directory, but are built into bindata.go so that Gomacs will work without needing
to install them anywhere. They are written in a simple yaml format.

The undo struct is based at least in part on the one in [the suckless editor Sandy.](http://tools.suckless.org/sandy)
However, most of the undo logic is custom.

The binary-tree model of window management comes from my hex editor,
[Zerz,](https://github.com/japanoise/zerz) and ultimately is inspired by i3's
and tmux's model of hsplits and vsplits.

## The main function

When starting Gomacs, we first get the global state started up, which includes
the lisp interpreter, get a little information from the command line arguments,
and open any requested files. This is done in main; opening files is done in a
fairly transparent way in EditorOpen(string). Once we have this information we
are ready to start termbox, defer our close function, and start the main loop.

The main loop is responsible for:

1. Refreshing the screen to the main editor view
2. Exiting if it has been told to
3. Getting the next key press
4. Either:
  - Sending the keypress' bound function to the lisp interpreter (or warning the
    user if it doesn't have one)
  - Quitting if the panic key was pressed
  - Inserting the UTF-8 encoded string sent by the last keypress, if it's only
    one rune long.

All very straightforward!

## Lisp hacking

[Make sure to read the Zygomys documentation.](https://github.com/glycerine/zygomys/wiki)
Note that Zygomys catches most crashes, and writes out the stacktrace to the
file crash.log in the current directory. If you get stuck (sometimes Zygomys will
crash and not get restarted) hit the `<f12>` key, which will quit Gomacs
immediately.

See the examples directory for some examples of lisp in use for extending
Gomacs. You can also find all of the functions that are currently implemented in
lisp.go. lispSingleton maps a simple func() to a function in lisp that takes no
arguments.

## Undo and Redo

Undo and redo structs are stored as stacks in each buffer. The same struct is
used for both undo and redo, as it merely encodes what was done in editor terms.

This code is a bit ropey since a lot of the editor functions take care of their
own undos. This could probably be made better by moving undo code to functions
actually called by user code (insertion, deletion, etc.)

The functions exposed to the user are editorDoRedo and editorDoUndo. These
similar functions act on the stacks in the current buffer, warning the user if
there's nothing to undo or redo.

Subsequent edits on the same line in adjacent positions are concatenated
together, as in GNU Emacs, to avoid making the undoing of long edits tedious.

## Region

Region commands are in region.go. They were rewritten recently to free them
from the helpful extra functionality in the editing commands, namely the
indentation and syntax highlighting updates - the latter of which they now
save until the end. These commands use their own logic for acting on the
buffer.

Region commands tend to take startc, startl, endc, endl. They assume that the
start numbers =< end numbers, and will probably fail spectacularly if this is
not the case.

The idioms used here are mostly emacs'. spitRegion is essentially a yank region
command.

## Input

The input functionality comes from my library
[termutil.](https://github.com/japanoise/termbox-util)

### Keys

Key commands are generally expected to be stored in a string in the usual emacs
format:

- `C-z`
- `M->`
- `C-x 0`
- `C-x C-c`
- `LEFT`

The difference between this format and GNU's is that we don't ever wrap commands
in `<>`.

### Getting a choice

Use the editorChoiceIndex function to get one of n choices from the user. This
command will take a default and a list of strings. The user's selection is
guaranteed to be a valid index of the array passed to this function unless they
cancel and the default value is not valid. This can be used to determine if the
user cancels, if you so desire.

Use the editorYesNoPrompt function to get a yes/no choice from the user. You can
optionally allow the user to cancel.

### Getting a string

Use editorPrompt to get a string from the user. You can also supply a callback
function to act on the string as it's typed; for an example of this see
editorFind and editorFindCallback.

## Commands

Commands are stored:

1. As a mapping between a string that represents a keymap and a function that
has access to the lisp environment. This behaves like a tree; some nodes contain
another map, some contain code. This allows us to specify keymaps like
`C-x C-s`.
2. As a string which names the command and a function with access to the lisp
environment. This allows us to name commands for use with `M-x` or `C-h c`.

It's strongly recommended that you define commands and then bind them to keys,
rather than using lisp functions, because `C-h c` can't access the lisp code.

## Minor Modes

Minor modes are stored as a simple map of strings to booleans, for fairly fast
access. They can be toggled with the command `toggle-mode`, and you can view the
current buffer's activated modes with the `show-modes` command. See the README
for the list of modes and what they do.
