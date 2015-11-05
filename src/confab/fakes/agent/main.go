package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	// store information about this fake process into JSON
	signal.Ignore()

	var outputData struct {
		Args []string
		PID  int
	}
	outputData.PID = os.Getpid()
	outputData.Args = os.Args[1:]
	outputBytes, err := json.Marshal(outputData)
	if err != nil {
		panic(err)
	}

	// validate command line arguments
	// expect them to look like
	//   fake-thing agent -config-dir=/some/path/to/some/dir
	if len(outputData.Args) == 0 {
		log.Fatal("expecting command as first argment")
	}
	var configDir string
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.StringVar(&configDir, "config-dir", "", "config directory")
	flagSet.Parse(outputData.Args[1:])
	if configDir == "" {
		log.Fatal("missing required config-dir flag")
	}

	// save information JSON to the config dir
	err = ioutil.WriteFile(filepath.Join(configDir, "fake-output.json"), outputBytes, 0600)
	if err != nil {
		panic(err)
	}

	// read input options provided to us by the test
	var inputOptions struct {
		Slow       bool
		WaitForHUP bool
	}
	if optionsBytes, err := ioutil.ReadFile(filepath.Join(configDir, "options.json")); err == nil {
		json.Unmarshal(optionsBytes, &inputOptions)
	}

	if inputOptions.Slow {
		time.Sleep(10 * time.Second)
	}
	if inputOptions.WaitForHUP {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
		}
	}

	fmt.Fprintf(os.Stdout, "some standard out")
	fmt.Fprintf(os.Stderr, "some standard error")
}
