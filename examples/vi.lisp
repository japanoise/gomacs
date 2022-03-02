// Example rc file - you have to press i before you can insert ;)
(adddefaultmode "no-self-insert-mode")
(emacsdefinecmd "vi-insert" setmode "no-self-insert-mode" false)
(emacsbindkey "i" "vi-insert")
