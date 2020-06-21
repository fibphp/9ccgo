package main

import (
	"encoding/json"
	"fmt"
	. "go9cc"
	"io/ioutil"
	"log"
	"os"
	. "utils"
)

type CfgConfig struct {
	Cmd     string                 `json:"cmd"`
	Zc      []string               `json:"zc"`
	Warn    map[string]string      `json:"warn"`
	Flag    []string               `json:"flag"`
	File    map[string]string      `json:"file"`
	Input   []string               `json:"input"`
	Include []string               `json:"include"`
	Define  map[string]interface{} `json:"define"`
}

type ClPayload struct {
	file   string
	tokens *Vector
}

func (p *ClPayload) Name() string {
	return p.file
}
func (p *ClPayload) Play(w *Worker) {
	input := p.file
	fmt.Printf("Info tokenize worker:%s <%d> file:%s \n", w.Name(), GetGID(), input)
	tokens := Tokenize(input, true, nil)
	fmt.Printf("Info done worker:%s <%d> file:%s tokens:%d \n", w.Name(), GetGID(), input, tokens.Len())
	p.tokens = tokens
}

func do_cl(cfgs []CfgConfig) {
	jobQueue := make(chan *Job)
	dispatch := NewDispatcher("cl", 8, jobQueue, false)
	dispatch.Run()

	for _, cfg := range cfgs {
		inputs := cfg.Input
		for _, input := range inputs {
			input = input + "ipp"
			if _, err := os.Stat(input); err == nil || os.IsExist(err) {
				jobQueue <- &Job{
					Payload: &ClPayload{
						file: input,
					},
				}
			} else {
				fmt.Printf("Error Not Found file:%s \n", input)
			}
		}
	}

	dispatch.Join()
	dispatch.Stop()
	close(jobQueue)
}

func main() {

	debug := false
	if len(os.Args) == 1 {
		usage()
	}
	if len(os.Args) == 2 && os.Args[1] == "-test" {
		Util_test()
		os.Exit(0)
	}

	if len(os.Args) == 2 && os.Args[1] == "-cl" {
		cfgFile := "nmake_cl.json"
		data, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			fmt.Printf("Error Read config nmake_cl.json.\n")
			log.Fatal(err)
		}

		var cfgList []CfgConfig
		if err := json.Unmarshal(data, &cfgList); err != nil {
			fmt.Printf("Error Parse config nmake_cl.json.\n")
			log.Fatal(err)
		}
		do_cl(cfgList)
		os.Exit(0)
	}

	path := ""
	dump_ir1 := false
	dump_ir2 := false

	if len(os.Args) == 3 && os.Args[1] == "-dump-ir1" {
		dump_ir1 = true
		path = os.Args[2]
	} else if len(os.Args) == 3 && os.Args[1] == "-dump-ir2" {
		dump_ir2 = true
		path = os.Args[2]
	} else {
		if len(os.Args) != 2 {
			usage()
		}
		path = os.Args[1]
	}

	// Tokenize and parse.
	tokens := Tokenize(path, true, nil)
	if debug {
		Print_tokens(tokens)
	}
	nodes := Parse(tokens)
	globals := Sema(nodes)
	fns := Gen_ir(nodes)

	if dump_ir1 {
		Dump_ir(fns)
	}

	Alloc_regs(fns)
	if dump_ir2 {
		Dump_ir(fns)
	}

	Gen_x86(globals, fns)
}

func usage() { ErrorReport("Usage: 9ccgo [-test] [-dump-ir1] [-dump-ir2] <file>") }
