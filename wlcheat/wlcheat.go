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

func modifyBlocks(blocks []msq.Block, dims []gen.Point) error {
	for i, _ := range blocks {
		b := &blocks[i]
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

		*b = m.Block()
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

	blocks0, blocks1, err := wlutil.ReadAndParseGames(inDir)
	if err != nil {
		onErr(err)
	}

	// Set random encounter chance to 0.
	if err := modifyBlocks(blocks0[:defs.Block0NumBlocks], defs.MapDims[0]); err != nil {
		panic(err.Error())
	}
	if err := modifyBlocks(blocks1[:defs.Block1NumBlocks], defs.MapDims[1]); err != nil {
		panic(err.Error())
	}

	// Set all PCs' attributes to 127.
	for i := 0; i < 4; i++ {
		blocks0[20].EncSection[i*0x100+0x10e] = 0x7f
		blocks0[20].EncSection[i*0x100+0x10f] = 0x7f
		blocks0[20].EncSection[i*0x100+0x110] = 0x7f
		blocks0[20].EncSection[i*0x100+0x111] = 0x7f
		blocks0[20].EncSection[i*0x100+0x112] = 0x7f
		blocks0[20].EncSection[i*0x100+0x113] = 0x7f
		blocks0[20].EncSection[i*0x100+0x114] = 0x7f
		blocks0[20].EncSection[i*0x100+0x11a] = 15

		// Fill up PCs' inventory.
		for j := 0; j < 50; j++ {
			blocks0[20].EncSection[i*0x100+0x1bd+j] = byte(j)
		}
	}

	if err := wlutil.SerializeAndWriteGames(blocks0, blocks1, inDir); err != nil {
		onErr(err)
	}
}
