# cclo

To cache the output of the command line

## Usage

```sh
# if first time then run the command and to cache
cclo -- echo "hello world"

# if next time then display cached outputs
cclo -- echo "hello world"

# output same times
cclo date; sleep 1; cclo date
```

## Installation

go get

## License

MIT
