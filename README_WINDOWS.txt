Running Gomacs on Windows
=========================

Gomacs was primarily designed with unix-like operating systems in mind. That's
not to say that it's unusable on Windows; just that Windows support isn't my
primary concern.

With that in mind, here's some stuff to help you run Gomacs on Windows.

rc file
=======

The rc file lives in %APPDATA%, which is usually:

    C:\Users\<User>\AppData\Roaming

in recent versions of Windows. The file is inside that directory, in:

    japanoise\gomacs\rc.zy

Terminal Title
==============

Windows doesn't support setting the terminal title, so terminal-title-mode does
nothing.

Control-H
=========

Windows uses ^H for backspace. There's already a workaround in termbox-util for
this, but for Gomacs it leaves us with the problem of what to do about the C-h
family of commands - very useful help functions.

In keeping with tradition for UI, especially on Windows, the help functions are
additionally mapped to F1, so wherever you see C-h in the documentation, replace
it with F1.
