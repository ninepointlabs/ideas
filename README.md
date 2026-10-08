# idea

A tiny CLI for jotting down numbered ideas in a markdown file. Go stdlib only.

## Install

```sh
make build              # builds ./idea
sudo make install       # installs to /usr/local/bin/idea
```

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
```

After removal, remaining ideas are renumbered from 1. If any number in the
spec is invalid or out of range, nothing is removed.

## Storage

Ideas live in `~/Documents/Ideas/ideas.md`. Override with `$IDEAS_FILE`:

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

Note: the file is rewritten on every add/remove, so only the `# Ideas` header
and numbered lines are preserved.