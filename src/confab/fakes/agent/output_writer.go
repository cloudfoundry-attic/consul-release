package main

import (
	"encoding/json"
	"io/ioutil"
)

type OutputWriter struct {
	filepath      string
	data          OutputData
	callCountChan chan string
}

type OutputData struct {
	Args                []string
	PID                 int
	LeaveCallCount      int
	UseKeyCallCount     int
	InstallKeyCallCount int
}

func NewOutputWriter(filepath string, pid int, args []string) *OutputWriter {
	ow := &OutputWriter{
		filepath: filepath,
		data: OutputData{
			PID:  pid,
			Args: args,
		},
		callCountChan: make(chan string),
	}

	go ow.run()

	return ow
}

func (ow *OutputWriter) run() {
	ow.writeOutput()
	for {
		switch <-ow.callCountChan {
		case "leave":
			ow.data.LeaveCallCount++
		case "installkey":
			ow.data.InstallKeyCallCount++
		case "usekey":
			ow.data.UseKeyCallCount++
		case "exit":
			return
		}
		ow.writeOutput()
	}
}

func (ow OutputWriter) writeOutput() {
	outputBytes, err := json.Marshal(ow.data)
	if err != nil {
		panic(err)
	}

	// save information JSON to the config dir
	err = ioutil.WriteFile(ow.filepath, outputBytes, 0600)
	if err != nil {
		panic(err)
	}
}

func (ow *OutputWriter) LeaveCalled() {
	ow.callCountChan <- "leave"
}

func (ow *OutputWriter) UseKeyCalled() {
	ow.callCountChan <- "usekey"
}

func (ow *OutputWriter) InstallKeyCalled() {
	ow.callCountChan <- "installkey"
}

func (ow *OutputWriter) Exit() {
	ow.callCountChan <- "exit"
}
