package main

import (
	"gocapsule/gocapsule"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(gocapsule.Analyzer)
}
