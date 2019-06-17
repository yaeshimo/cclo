package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cache directories priority:
//	1. $XDG_CACHE_HOME/cclo/COMMANDNAME.json
//	2. ~/.cache/cclo/COMMANDNAME.json
//
// consider use cclo/ to cclo/cache/

// TODO: move making directory function to other places
var cachedir = func() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	dir = filepath.Join(dir, "cclo")
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			panic(err)
		}
		return dir
	}
	if !fi.IsDir() {
		panic("seems not directory " + dir)
	}
	return dir
}()

// impl:
// - parse command line
// - read cache
// - if cached then display a cache
// - else, to run the commands
// - if not cached then to cache a outputs

// consider options:
// -f Force to run
// -l List caches
// -r, --raw Raw outputs
// -remove COMMAND Remove specific commands caches
// -remove COMMAND ARGUMENTS... Remove a caches

// data design for cache
// pick data formats
// use sql?
// requred contents:
//	1. cached date
//	2. raw outputs
//	3. runned command line

// json
// filename is same that about target command names
type Cache struct {
	// []string{"cmdpath", "arg1", "arg2", ...}
	Args   []string  `json:"Args"`
	Date   time.Time `json:"date"`
	Output []byte    `json:"output"`
}

// json
type Caches struct {
	// path to cache file
	path string

	// command name
	Cmd string `json:"cmd"`

	// map[strings.Join(Args, " ")]Cache
	Caches map[string]Cache `json:"caches"`
}

// cache filename is cmdpath + ".json"
func ReadCache(cmdpath string) (*Caches, error) {
	name := filepath.Base(cmdpath)
	cs := &Caches{
		path:   filepath.Join(cachedir, name+".json"),
		Cmd:    name,
		Caches: make(map[string]Cache),
	}
	b, err := ioutil.ReadFile(cs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return cs, nil
		}
		return nil, err
	}
	err = json.Unmarshal(b, cs)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

func (cs *Caches) WriteCache() error {
	b, err := json.MarshalIndent(cs, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cs.path, b, 0600)
}

// args[0] is path to cmd, args[1:] are arguments
// not cache stderr, caches only stdout
// if force is true then force to run
func runcmd(stdout, stderr io.Writer, stdin io.Reader, args []string, force bool) error {
	if len(args) < 1 {
		return errors.New("command not specified")
	}
	cs, err := ReadCache(args[0])
	if err != nil {
		return err
	}

	key := strings.Join(args, " ")

	// use cache
	if !force {
		cache, ok := cs.Caches[key]
		if ok {
			_, err := fmt.Fprint(stdout, string(cache.Output))
			return err
		}
	}

	buf := new(bytes.Buffer)
	mw := io.MultiWriter(stdout, buf)

	// run commands
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = mw
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	err = cmd.Run()
	if err != nil {
		return err
	}

	// to cache
	// TODO: reconsider
	// needs at here?
	cs.Caches[key] = Cache{
		Args:   args,
		Date:   time.Now(),
		Output: buf.Bytes(),
	}
	return cs.WriteCache()
}

// if cmd is "" then print only cached command names
// if specified command name then print caches
func List(w io.Writer, cmd string) error {
	fis, err := ioutil.ReadDir(cachedir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if !strings.HasSuffix(fi.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(fi.Name()), ".json")
		if cmd == "" {
			// list only command names
			_, err := fmt.Fprintln(w, name)
			if err != nil {
				return err
			}
		} else {
			// list cached arguments
			if cmd == name {
				caches, err := ReadCache(cmd)
				if err != nil {
					return err
				}
				for _, cache := range caches.Caches {
					// TODO: formats
					_, err := fmt.Fprintf(w, "%v: %q\n", cache.Date, cache.Args)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

const (
	Name    = `cclo`
	Version = `0.1.0`
)

const usage = `Description:
  To cache the output of the command line

Usage:
  cclo [Options]
  cclo [Options] [--] COMMAND ARGUMENTS...

Options:
  -help         Display this message
  -version      Display version
  -list COMMAND List cached commands

Examples:
  $ cclo -help # display help
  $ cclo date; sleep 1; cclo date # output same times
  $ cclo -list # list cached command names
  $ cclo -list # list specific commands caches
`

var usageWriter io.Writer = os.Stderr

var opt struct {
	help    bool
	version bool
	list    bool

	force bool
}

func init() {
	flag.BoolVar(&opt.help, "help", false, "Display this message")
	flag.BoolVar(&opt.version, "version", false, "Display version")
	flag.BoolVar(&opt.list, "list", false, "List cached commands")

	flag.BoolVar(&opt.force, "force", false, "Ignore cache and force to run")
	flag.BoolVar(&opt.force, "f", false, "Alias of -force")

	flag.Usage = func() { fmt.Fprintln(usageWriter, usage) }
}

func run() error {
	flag.Parse()
	switch {
	case opt.help:
		usageWriter = os.Stdout
		flag.Usage()
		return nil
	case opt.version:
		_, err := fmt.Printf("%s %s\n", Name, Version)
		return err
	}
	if flag.NArg() < 1 && !opt.list {
		flag.Usage()
		return errors.New("command not specified")
	}
	if opt.list {
		switch flag.NArg() {
		case 0:
			return List(os.Stdout, "")
		case 1:
			return List(os.Stdout, flag.Arg(0))
		default:
			return errors.New("too many arguments")
		}
	}
	return runcmd(os.Stdout, os.Stderr, os.Stdin, flag.Args(), opt.force)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
