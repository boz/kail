package main

import (
	"fmt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	flagNs    = kingpin.Flag("ns", "namespace").Strings()
	flagPod   = kingpin.Flag("pod", "pod").Strings()
	flagSvc   = kingpin.Flag("svc", "service").Strings()
	flagNode  = kingpin.Flag("node", "node").Strings()
	flagLabel = kingpin.Flag("label", "label").PlaceHolder("NAME=VALUE").Strings()

	flagDryRun = kingpin.Flag("dry-run", "print matching pods and exit").Bool()
)

func main() {
	fmt.Println("vim-go")
}
