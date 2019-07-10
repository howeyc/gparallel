package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/facebookgo/limitgroup"
	"github.com/mattn/go-shellwords"
	"gopkg.in/cheggaaa/pb.v1"
)

func main() {
	var quiet, verbose bool
	var limit uint
	flag.BoolVar(&quiet, "q", false, "quiet: supress stderr")
	flag.BoolVar(&verbose, "v", false, "verbose: show stdout")
	flag.UintVar(&limit, "j", 4, "jobs: number of concurrent jobs")
	flag.Parse()
	commandline := strings.Join(flag.Args(), " ")

	conc := limitgroup.NewLimitGroup(limit)

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	bar := pb.StartNew(len(lines))

	var exitcode int

	for i, line := range lines {
		rcommand := strings.ReplaceAll(commandline, "{}", line)
		rcommand = strings.ReplaceAll(rcommand, "{#}", fmt.Sprint(i))
		args, perr := shellwords.Parse(rcommand)
		if perr != nil {
			fmt.Println(perr, line)
		} else {
			conc.Add(1)
			go func(args []string) {
				cmd := args[0]
				rest := args[1:]
				c := exec.Command(cmd, rest...)
				var buf, ebuf bytes.Buffer
				c.Stdout = &buf
				c.Stderr = &ebuf
				err := c.Run()
				if err != nil {
					if exerr, ok := err.(*exec.ExitError); ok {
						exitcode = exerr.ExitCode()
					}
					fmt.Fprintln(os.Stderr, err)
				}
				if verbose {
					io.Copy(os.Stdout, &buf)
				}
				if !quiet {
					io.Copy(os.Stderr, &ebuf)
				}
				conc.Done()
				bar.Increment()
			}(args)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	conc.Wait()
	bar.FinishPrint("")

	os.Exit(exitcode)

	return
}

