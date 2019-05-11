package main

import (
	"flag"
	"fmt"
	"github.com/wsxiaoys/terminal/color"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Configuration info (host config(s) + cli args)
//
type Configuration struct {
	Hostfile string
	Hostlist []*HostConfig
	Tag      string
	Timeout  int
	Cmd      string
}

// Config for a single host (eg. a single line in the config)
//
type HostConfig struct {
	Username    string
	Host        string
	Credentials string
	Tags        []string // optional
}

// determine config file to read
//
func determineFile(hostflag string) string {
	for _, filename := range []string{hostflag, os.Getenv("MULTICMD_HOSTS")} {
		if filename != "" {
			return filename
		}
	}
	return "hosts.ini"
}

// Parse all the things and return Configuration information, or die trying
//
func parseArgs() *Configuration {
	conf := &Configuration{}

	hostlstPtr := flag.String("f", "", "Host list [hosts.ini] (env var: MULTICMD_HOSTS)")
	tagPtr := flag.String("t", "", "Tag. All hosts with the given Tag will have the cmd run on them.")
	timeoutPtr := flag.Int("timeout", -1, "timeout where -1 is 'no timeout' [-1]")

	flag.Parse()

	conf.Cmd = strings.Join(flag.Args(), " ")
	conf.Tag = *tagPtr
	conf.Hostfile = determineFile(*hostlstPtr)
	conf.Timeout = *timeoutPtr
	conf.Hostlist = obtainHostlist(conf.Hostfile, []string{conf.Tag})

	if len(conf.Hostlist) < 1 {
		log.Fatalln("No matching hosts found in host list", conf.Hostfile)
	}

	return conf
}

// Write some output from a host to the terminal.
//  - Handle stdout/stderr and apply some snazzy colours (yay).
//
func logline(cmd *SshCmdRunner) (int, error, int, error) {
	read, errOne := cmd.Stdout()
	stdoutLen := len(read)
	if errOne == nil && stdoutLen > 0 {
		line := string(read)
		if line != "" && line != "\n" {
			color.Printf("@{!g}[out] %s @{!b}%s\n", cmd.Host, line)
		}
	}

	read, errTwo := cmd.Stderr()
	stderrLen := len(read)
	if errTwo == nil && len(read) > 0 {
		line := string(read)
		if line != "" && line != "\n" {
			color.Printf("@{!r}[err] %s @{!b}%s\n", cmd.Host, line)
		}
	}

	return stdoutLen, errOne, stderrLen, errTwo
}

func main() {
	config := parseArgs()

	var wg sync.WaitGroup

	running := make(chan *SshCmdRunner)
	cmds := []*SshCmdRunner{}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGABRT)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGKILL)

	go func() { // a routine to check for incoming signals from the user
		<-c
		fmt.Println("[signal caught] terminating")

		running <- nil
		close(running)

		for _, cmd := range cmds {
			cmd.Kill()
		}

		os.Exit(1)
	}()

	for _, hostline := range config.Hostlist { // connect to each host & fire commands
		wg.Add(1)
		hostData := hostline

		go func() {
			c, err := NewSshCmdRunner(hostData.Host, hostData.Username, hostData.Credentials)
			if err != nil {
				fmt.Println(err)
				wg.Done()
				return
			}
			running <- c

			if config.Timeout > 0 { // if a time out was set, make sure to trigger this
				go func() {
					time.Sleep(time.Second * time.Duration(config.Timeout))
					c.Kill()
					color.Printf("@{!r}[err] %s [timeout] SIGABRT sent\n", c.Host)
				}()
			}

			err = c.Execute(config.Cmd)
			if err != nil {
				fmt.Println(err)
			}
			wg.Done()
		}()
	}

	go func() { // routine to print cmds
		// This is essentially passed all running commands to poll for output
		for {
			select {
			case c := <-running:
				if c == nil { // break when we get sent a nil
					break
				}

				cmds = append(cmds, c) // otherwise we'll manage this cmd too
			default:
				for _, cmd := range cmds {
					logline(cmd) // try to print everything, we don't care about errors
				}
			}

		}
	}()

	wg.Wait()      // wait for cmds to finish
	running <- nil // send poison pill to printing routine (we'll take over to flush & exit)
	close(running)

	for _, cmd := range cmds { // flush before closing
		// if at least one of the buffers is still printing & hasn't errored out, keep printing
		ro, stdouterr, rt, stderrerr := logline(cmd)
		for (stdouterr == nil || stderrerr == nil) && ro+rt > 0 {
			ro, stdouterr, rt, stderrerr = logline(cmd)
		}

		cmd.Kill() // if the cmd isn't done by now, we're sending the ABORT signal.
	}
}
