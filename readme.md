<h1 align="center">webout</h1>

Pipe terminal output to the browser and see it in real time.

## Install

```
go get -u github.com/gillchristian/webout/cmd/webout
```

## Usage

```
$ webout --help
NAME:
   webout - Pipe terminal output to the browser

USAGE:
   $ webout ping google.com

VERSION:
   0.0.1

AUTHOR:
   Christian Gill (gillchristiang@gmail.com)

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

## TODO

**CLI**

- [ ] Allow piping `stdin` to `webout` CLI
- [ ] Opt-out of piping `stderr` on controlled mode
- [ ] Silent mode
- [ ] Copy channel URL to clipboard when created ([github.com/atotto/clipboard](https://github.com/atotto/clipboard))
- [ ] Open channel URL in browser when created

**Server**

- [ ] Persist channels & content
- [ ] Limit the number of in-memory channels
- [ ] Add users with GitHub OAuth
- [ ] Make autoscroll toggle-able
- [ ] Better home (animate the example & add "browser" showing channel)
- [ ] Overview of my channels
