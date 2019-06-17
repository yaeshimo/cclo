# cclo

To cache the output of the command line

## Usage

```sh
# if first time then run the command and to cache
cclo -- date

# if next time then display cached outputs
cclo -- date
```

If you use `pacman` then you can cache the `checkupdates`

```sh
# require before run
#pacman -S pacman-contrib

# force to run the checkupdates and to cache
cclo -f -- checkupdates

# display cached outputs
cclo -- checkupdates

# example of alias
alias checkupdates='cclo -f -- checkupdates'
alias checklog='cclo -- checkupdates'
```

## Requirements

User cache directory

### Linux

`$XDG_CACHE_HOME/cclo/` or `~/.cache/cclo/`

## Installation

go get

## License

MIT
