# mved

Move/rename/delete files and directories using $EDITOR!

## Why

Sometimes you want to rename multiple files, delete based on some regex, split a directory, and so on. This is annoying to to using normal command line tools.

`mved` solves this by moving the work to `$EDITOR`, allowing you to do this in the comfort of your prefered text-editor.

moving mulitple files, renaming them, deleting some, all easy and doable in on go.

## How

install the binary using `go install`

```
go install github.com/fredrikkvalvik/mved
```

run `mved -h` for help.

run `mved` for the basic usecase. This runs the tool in `$CWD`.

To move/rename a file, simply change the name of the file on the line.

> **NOTE**
> the number at the start of each line is the ID of the file. This is important to keep in mind.

To delete a file, remove the line from the text buffer (of comment it out by adding `#` at the start of the line)

## How

`mved` read the current directory, creates a list of entries, and outputs that to a file that you edit.
When the editor closed, we parse the new file, validate it to make sure that it parses correctly and does not contain errors.
If the parsed file is valid, we create a dependancy graph and resolve the order in which the changes to the filesystem happen.

As long as the graph contains no circular dependancies, the changes are made.
