# idea

`idea` (from the `ideas` repo) is a tiny CLI for jotting down numbered ideas in a markdown file. Go stdlib only.

## Install

With [autonomix-cli](https://github.com/ninepointlabs/autonomix-cli):

```sh
autonomix-cli add https://github.com/ninepointlabs/ideas
```

Or download the `.deb`, `.rpm` or Arch package from the
[latest release](https://github.com/ninepointlabs/ideas/releases/latest).

From source:

```sh
make build                         # builds ./idea
make install PREFIX="$HOME/.local" # installs to ~/.local/bin/idea, no sudo
sudo make install                  # or system-wide, to /usr/local/bin/idea
```

`~/.local/bin` must be on your `PATH`.

## Usage

```sh
idea "Build a CLI for ideas"     # append an idea
idea Multiple words work too     # args are joined with spaces
idea -- "-starts with a dash"    # use -- for text beginning with '-'

idea -l                          # list all ideas

idea -r 1                        # remove idea 1
idea -r 1-3                      # remove ideas 1 through 3
idea -r 1,5,6,9                  # remove specific ideas
idea -r 1-3,5,7-9                # mix ranges and numbers

idea --version                   # show the version
```

After removal, remaining ideas are renumbered from 1. If any number in the
spec is invalid or out of range, nothing is removed.

## Storage

Ideas live in `~/Documents/Ideas/ideas.md`. Override with `$IDEAS_FILE`
(its folder is created if it doesn't exist):

```sh
IDEAS_FILE=~/notes/work-ideas.md idea "Ship it"
```

Format:

```markdown
# Ideas
1. First idea
2. Second idea
3. Third idea
```

Numbered lines are ideas. Any other lines, such as headings or notes, are kept
as they are when you add or remove ideas; the remaining ideas are renumbered
in place. A read-only ideas file is left alone: `idea` exits with an error
instead of changing it.