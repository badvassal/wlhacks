package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/badvassal/wllib/decode"
	"github.com/badvassal/wllib/defs"
	"github.com/badvassal/wllib/gen"
	"github.com/badvassal/wllib/modify"
	"github.com/badvassal/wllib/msq"
	"github.com/badvassal/wllib/wlutil"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: wlcheat <wl-dir>\n")
}

func onErr(err error) {
	fmt.Fprintf(os.Stderr, "* error: %s\n", err.Error())
	os.Exit(2)
}

func modifyBlocks(bodies []msq.Body, dims []gen.Point) error {
	for i, _ := range bodies {
		b := &bodies[i]
		dim := dims[i]

		db, err := decode.DecodeBlock(*b, dim)
		if err != nil {
			return err
		}

		db.MapInfo.EncounterFreq = 0

		m := modify.NewBlockModifier(*b, dim)
		if err := m.ReplaceMapInfo(db.MapInfo); err != nil {
			return err
		}

		*b = m.Body()
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	log.SetLevel(log.WarnLevel)

	inDir := os.Args[1]

	descs0, descs1, err := wlutil.ReadAndParseGames(inDir)
	if err != nil {
		onErr(err)
	}

	bodies0 := wlutil.DescsToBodies(descs0)
	bodies1 := wlutil.DescsToBodies(descs1)

	// Set random encounter chance to 0.
	if err := modifyBlocks(bodies0[:defs.Block0NumBlocks], defs.MapDims[0]); err != nil {
		panic(err.Error())
	}
	if err := modifyBlocks(bodies1[:defs.Block1NumBlocks], defs.MapDims[1]); err != nil {
		panic(err.Error())
	}

	// Set all PCs' attributes to 127.
	for i := 0; i < 4; i++ {
		bodies0[20].EncSection[i*0x100+0x10e] = 0x7f
		bodies0[20].EncSection[i*0x100+0x10f] = 0x7f
		bodies0[20].EncSection[i*0x100+0x110] = 0x7f
		bodies0[20].EncSection[i*0x100+0x111] = 0x7f
		bodies0[20].EncSection[i*0x100+0x112] = 0x7f
		bodies0[20].EncSection[i*0x100+0x113] = 0x7f
		bodies0[20].EncSection[i*0x100+0x114] = 0x7f
		bodies0[20].EncSection[i*0x100+0x11a] = 15
		bodies0[20].EncSection[i*0x100+0x120] = 0x7f

		// Fill up PCs' inventory.
		for j := 0; j < 50; j++ {
			bodies0[20].EncSection[i*0x100+0x1bd+j] = byte(j)
		}
	}

	if err := wlutil.SerializeAndWriteGames(bodies0, bodies1, inDir); err != nil {
		onErr(err)
	}
}
