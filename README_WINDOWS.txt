Running Gomacs on Windows
=========================

Gomacs was primarily designed with unix-like operating systems in mind. That's
not to say that it's unusable on Windows; just that Windows support isn't my
primary concern.

With that in mind, here's some stuff to help you run Gomacs on Windows.

rc file
=======

The rc file lives in either:

C:\Users\<Your name>\.gomacs.lisp

or:

C:\Documents and Settings\<Your name>\.gomacs.lisp

depending on your version of Windows. For now you can open it in Notepad.

Terminal Title
==============

Windows doesn't support setting the terminal title, so disable it in your rc
file:

(remdefaultmode "terminal-title-mode")

Control-H
=========

Windows uses ^H for backspace. There's already a workaround in termbox-util for
this, but for Gomacs it leaves us with the problem of what to do about the C-h
family of commands - very useful help functions.

This snippet from the examples should help in this usecase, by mapping the help
functions to <f1> instead of ^H:

(emacsbindkey "f1 a" "apropos-command")
(emacsbindkey "f1 c" "describe-key-briefly")
(emacsbindkey "f1 m" "show-modes")
(emacsbindkey "f1 b" "describe-bindings")
(emacsbindkey "f1 f1" "quick-help")
