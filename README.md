# Search-Directory-History

This is born from my switch to using the Oh-My-Zsh plugin for directory history,
[per-directory-history](https://github.com/jimhester/per-directory-history).  Initially I was satisfied with a simple function to grep through the .directory_history to find historical commands:

```bash
function sdh() { find ~/.directory_history/**/history -exec grep "$@" {} + | cut -d';' -f2 | sort | uniq; }
```

Sometimes I would find that I might want to know the source directory of where I ran those commands, so I could get some additional context of the commands just prior and just after.  Context is what keeps me using directory history, and I suspect was a driver for the initial concept.  So I've jotted down a few things that I would like a searching tool to do, and started creating `search-directory-history`.

## Features Map

- [x] Basic Search - return commands from a single keyword search
- [x] Intermediate Search - & and | for multi-keyword searching
- [ ] Advanced RegEx Searching - return commands matching a passed in regex
- [ ] Output - terse/verbose/specified fields

## Cobra Command Structure

- [Cobra GoLang Module/Tool Walkthrouh](https://www.linode.com/docs/development/go/using-cobra/)

### Commands: verbs, nouns adjectives

- search
  - Verbs
    - [x] `--multiline` - show entire multi-line commands
    - [x] `--startpath` - start the search at a specific path: `~/.directory_history/home/myuid/src/`
      - specified as `/home/myuid/src`
    - [ ] `--searchfrom` - timeframe in 4d, 3w, 2m, 1y
    - [ ] `--searchduration` - timeframe in 1d, 1w, 1m, 1y, starting from `--searchfrom`
      - defaults to `--searchfrom`
    - [ ] `--terse`
    - [ ] `--verbose`
    - [ ] `--fields` - csv list of field names `"date,directory,command"`
    - [ ] `--output` - json,table,text,yaml?
    - [ ] `--context` - number of context commands to display on either side of match
- help
