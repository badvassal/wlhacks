package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/badvassal/wllib/decode"
	"github.com/badvassal/wllib/defs"
	"github.com/badvassal/wllib/digest"
	"github.com/badvassal/wllib/gen"
	"github.com/badvassal/wllib/gen/wlerr"
	"github.com/badvassal/wllib/msq"
	"github.com/badvassal/wllib/wlutil"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: wldec <wl-dir> <out-dir>\n")
}

func onErr(err error) {
	fmt.Fprintf(os.Stderr, "* error: %s\n", err.Error())
	os.Exit(2)
}

func dumpJson(obj interface{}, filename string) error {
	j, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return wlerr.Wrapf(err, "failed to marshal json")
	}

	if err := ioutil.WriteFile(filename, []byte(j), 0644); err != nil {
		return wlerr.Wrapf(err, "failed to write json to disk")
	}

	return nil
}

func calcOutSubdir(gameIdx int, blockIdx int) string {
	return fmt.Sprintf("/g%db%02d", gameIdx, blockIdx)
}

func dumpCentralDir(cd decode.CentralDir, outDir string) error {
	return dumpJson(cd, outDir+"/centraldir.json")
}

func dumpRawData(block msq.Block, outDir string) error {
	encfn := outDir + "/encsection.bin"
	if err := ioutil.WriteFile(encfn, block.EncSection, 0644); err != nil {
		return wlerr.Wrapf(err, "failed to write enc section to disk")
	}

	plnfn := outDir + "/plainsection.bin"
	if err := ioutil.WriteFile(plnfn, block.PlainSection, 0644); err != nil {
		return wlerr.Wrapf(err, "failed to write plain section to disk")
	}

	return nil
}

func dumpSizes(block msq.Block, dim gen.Point, outDir string) error {
	cb, err := decode.CarveBlock(block, dim)
	if err != nil {
		return err
	}

	if err := dumpJson(cb.Sizes(), outDir+"/sizes.json"); err != nil {
		return err
	}

	return nil
}

func dumpOffsets(block msq.Block, dim gen.Point, outDir string) error {
	cb, err := decode.CarveBlock(block, dim)
	if err != nil {
		return err
	}

	if err := dumpJson(cb.Offsets, outDir+"/offsets.json"); err != nil {
		return err
	}

	return nil
}

func dumpMeta(block msq.Block, outDir string) error {
	if err := dumpJson(block, outDir+"/meta.json"); err != nil {
		return err
	}

	return nil
}

func dumpBlock(block msq.Block, gameIdx int, blockIdx int, outDir string) error {
	decErr := func(err error) error {
		return wlerr.Wrapf(err,
			"failed to decode block %d,%d", gameIdx, blockIdx)
	}

	outDir += "/" + calcOutSubdir(gameIdx, blockIdx)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return decErr(err)
	}

	if err := dumpMeta(block, outDir); err != nil {
		return decErr(err)
	}

	dim := defs.MapDims[gameIdx][blockIdx]
	db, err := decode.DecodeBlock(block, dim)
	if err != nil {
		return decErr(err)
	}

	if err := dumpRawData(block, outDir); err != nil {
		return decErr(err)
	}

	if err := dumpOffsets(block, dim, outDir); err != nil {
		return decErr(err)
	}

	if err := dumpSizes(block, dim, outDir); err != nil {
		return decErr(err)
	}

	mds := digest.MapDataString(db.MapData)
	mdfilename := fmt.Sprintf("%s/mapdata.txt", outDir)
	if err := ioutil.WriteFile(mdfilename, []byte(mds), 0644); err != nil {
		return decErr(err)
	}

	if err := dumpCentralDir(db.CentralDir, outDir); err != nil {
		return decErr(err)
	}

	if err := dumpJson(db.MapInfo, outDir+"/mapinfo.json"); err != nil {
		return decErr(err)
	}

	if err := dumpJson(db.ActionTables.Transitions,
		outDir+"/transitions.json"); err != nil {

		return decErr(err)
	}

	if err := dumpJson(db.ActionTables.Loots,
		outDir+"/loots.json"); err != nil {

		return decErr(err)
	}

	if err := dumpJson(db.MonsterNames, outDir+"/monsternames.json"); err != nil {
		return decErr(err)
	}

	if err := dumpJson(db.MonsterData, outDir+"/monsterdata.json"); err != nil {
		return decErr(err)
	}

	if err := dumpJson(db.StringsArea, outDir+"/stringsarea.json"); err != nil {
		return decErr(err)
	}

	dss, err := digest.DecompressStringsArea(db.StringsArea)
	if err != nil {
		return decErr(err)
	}
	var ss []string
	for _, ds := range dss {
		ss = append(ss, string(ds))
	}

	if err := dumpJson([]string(ss), outDir+"/strings.json"); err != nil {
		return decErr(err)
	}

	return nil
}

func partialDump(block msq.Block, gameIdx int, blockIdx int, outDir string) error {
	decErr := func(err error) error {
		return wlerr.Wrapf(err,
			"failed to decode block %d,%d", gameIdx, blockIdx)
	}

	outDir += "/" + calcOutSubdir(gameIdx, blockIdx)

	dim := defs.MapDims[gameIdx][blockIdx]

	if err := dumpRawData(block, outDir); err != nil {
		return decErr(err)
	}

	if err := dumpOffsets(block, dim, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "partial dump failed: %s\n", err.Error())

		fmt.Fprintf(os.Stderr, "attempting a minimal central directory dump\n")
		off := decode.MapDataLen(dim)
		blob, err := gen.ExtractBlob(block.EncSection, off,
			off+decode.CentralDirLen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "minimal central directory dump failed\n")
			return decErr(err)
		}

		cd, err := decode.DecodeCentralDir(blob)
		if err != nil {
			return decErr(err)
		}

		if err := dumpCentralDir(*cd, outDir); err != nil {
			return decErr(err)
		}

		return nil
	}

	cb, err := decode.CarveBlock(block, defs.MapDims[gameIdx][blockIdx])
	if err != nil {
		return decErr(err)
	}

	cd, err := decode.DecodeCentralDir(cb.CentralDir)
	if err != nil {
		return decErr(err)
	}

	if err := dumpCentralDir(*cd, outDir); err != nil {
		return decErr(err)
	}

	return nil
}

func dumpGame(blocks []msq.Block, gameIdx int, outDir string) {
	for i, block := range blocks {
		if err := dumpBlock(block, gameIdx, i, outDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to dump block: %s\n", err.Error())

			fmt.Fprintf(os.Stderr, "attempting a partial dump\n")
			if err := partialDump(block, gameIdx, i, outDir); err != nil {
				fmt.Fprintf(os.Stderr, "partial dump failed: %s\n", err.Error())
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	log.SetLevel(log.InfoLevel)

	inDir := os.Args[1]
	outDir := os.Args[2]

	blocks0, blocks1, err := wlutil.ReadAndParseGames(inDir)
	if err != nil {
		onErr(err)
	}

	dumpGame(blocks0[:defs.Block0NumBlocks], 0, outDir)
	dumpGame(blocks1[:defs.Block1NumBlocks], 1, outDir)

	for i := defs.Block0NumBlocks; i < len(blocks0); i++ {
		destDir := outDir + "/" + calcOutSubdir(0, i)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			onErr(err)
		}
		if err := dumpRawData(blocks0[i], destDir); err != nil {
			onErr(err)
		}
	}
	for i := defs.Block1NumBlocks; i < len(blocks1); i++ {
		destDir := outDir + "/" + calcOutSubdir(1, i)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			onErr(err)
		}
		if err := dumpRawData(blocks1[i], destDir); err != nil {
			onErr(err)
		}
	}
}
