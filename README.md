# ideas

A tiny CLI for jotting down numbered ideas in a markdown file. Go stdlib only.

## Install

```sh
make build              # builds ./ideas
sudo make install       # installs to /usr/local/bin/ideas
```

## Usage

```sh
ideas "Build a CLI for ideas"     # append an idea
ideas Multiple words work too     # args are joined with spaces
ideas -- "-starts with a dash"    # use -- for text beginning with '-'

ideas -l                          # list all ideas

ideas -r 1                        # remove idea 1
ideas -r 1-3                      # remove ideas 1 through 3
ideas -r 1,5,6,9                  # remove specific ideas
ideas -r 1-3,5,7-9                # mix ranges and numbers
```

After removal, remaining ideas are renumbered from 1. If any number in the
spec is invalid or out of range, nothing is removed.

## Storage

Ideas live in `~/.ideas.md`. Override with `$IDEAS_FILE`:

```sh
IDEAS_FILE=~/notes/work-ideas.md ideas "Ship it"
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
