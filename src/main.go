package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	file string
	tokens *Vector
}

func (p *ClPayload) Name() string {
	return p.file
}
func (p *ClPayload) Play(w *Worker) {
	input := p.file
	fmt.Printf("Info tokenize worker:%s <%d> file:%s \n", w.name, getGID(), input)
	tokens := tokenize(input, true, nil)
	fmt.Printf("Info done worker:%s <%d> file:%s tokens:%d \n", w.name, getGID(), input, tokens.len)
	p.tokens = tokens
}

func do_cl(cfgs []CfgConfig) {
	i := "Zend\\zend_alloc.cipp"
	tokenize(i, true, nil)

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
		util_test()
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
	tokens := tokenize(path, true, nil)
	if debug {
		print_tokens(tokens)
	}
	nodes := parse(tokens)
	globals := sema(nodes)
	fns := gen_ir(nodes)

	if dump_ir1 {
		dump_ir(fns)
	}

	alloc_regs(fns)
	if dump_ir2 {
		dump_ir(fns)
	}

	gen_x86(globals, fns)
}

func usage() { errorReport("Usage: 9ccgo [-test] [-dump-ir1] [-dump-ir2] <file>") }
