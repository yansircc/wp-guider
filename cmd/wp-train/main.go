package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "init":
		cmdInit()
	case "status":
		cmdStatus()
	case "next":
		cmdNext(args)
	case "verify":
		cmdVerify()
	case "progress":
		cmdProgress()
	case "snapshot":
		cmdSnapshot()
	case "history":
		cmdHistory(args)
	case "inject":
		cmdInject(args)
	case "checkpoint":
		cmdCheckpoint(args)
	case "explain":
		if len(args) < 1 {
			fatal("usage: wp-train explain <topic>")
		}
		cmdExplain(args[0])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `wp-train: WordPress training CLI

Commands:
  init                Create/reset training site
  status              Show current training status (JSON)
  next [--topic=X]    Get next task (--force to skip current)
  verify              Verify current task completion
  progress            Show formatted progress
  snapshot            Full site state snapshot (JSON)
  history [-n 10]     Recent attempt history (JSON)
  inject [type]       Inject a fault for troubleshooting training
  checkpoint <cmd>    Save/restore/list site checkpoints
  explain <topic>     Explain a topic (e.g. L1.1)`)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
