# plugin

The Omarchy 4 shell overlay. Not started.

It will draw the board from `qmk info`'s physical key coordinates and drive
the CLI, which is why the CLI addresses keys by index: the ergonomics belong
here, not in the terminal.

This directory exists so release-please has a path to route `feat(plugin):`
commits to. It versions separately from `cli`.
