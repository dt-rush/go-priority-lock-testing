package main

import (
	"fmt"
	"go.uber.org/atomic"
)

func Log(s string, args ...interface{}) {
	if !(*profile) {
		if len(args) == 0 {
			fmt.Println(s)
		} else {
			fmt.Printf(s, args...)
		}
	}
}

func VerboseLog(s string, args ...interface{}) {
	if *verbose {
		Log(s, args...)
	}
}

type IDGenerator struct {
	x atomic.Uint32
}

func (g *IDGenerator) Gen() int {
	return int(g.x.Inc())
}

var IDGEN_OBJ = IDGenerator{}
var IDGEN = IDGEN_OBJ.Gen
