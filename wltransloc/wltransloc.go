package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/structs"
	log "github.com/sirupsen/logrus"

	"github.com/badvassal/wllib/decode"
	"github.com/badvassal/wllib/decode/action"
	"github.com/badvassal/wllib/defs"
	"github.com/badvassal/wllib/gen"
	"github.com/badvassal/wllib/gen/wlerr"
	"github.com/badvassal/wllib/msq"
	"github.com/badvassal/wllib/wlutil"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: wltransloc <wl-dir>\n")
}

func onErr(err error) {
	fmt.Fprintf(os.Stderr, "* error: %s\n", err.Error())
	printUsage()
	os.Exit(2)
}

type TransMark struct {
	Coords   []gen.Point
	Selector int
}

func findTransitions(db decode.Block) []TransMark {
	m := map[int][]gen.Point{}

	for y := 0; y < db.Dim.Y; y++ {
		for x := 0; x < db.Dim.X; x++ {
			c := db.MapData.ActionClasses[y][x]
			s := db.MapData.ActionSelectors[y][x]

			if c == action.IDTransition {
				m[s] = append(m[s], gen.Point{x, y})
			}
		}
	}

	var tms []TransMark
	for s, coords := range m {
		tms = append(tms, TransMark{
			Coords:   coords,
			Selector: s,
		})
	}

	for i, _ := range db.ActionTables.Transitions {
		found := false
		for _, tm := range tms {
			if tm.Selector == i {
				found = true
				break
			}
		}

		if !found {
			tms = append(tms, TransMark{
				Coords:   nil,
				Selector: i,
			})
		}
	}

	sort.Slice(tms, func(i int, j int) bool {
		return tms[i].Selector < tms[j].Selector
	})

	return tms
}

func dumpBlock(body msq.Body, dim gen.Point) ([]map[string]interface{}, error) {
	db, err := decode.DecodeBlock(body, dim)
	if err != nil {
		return nil, err
	}

	var ms []map[string]interface{}

	tms := findTransitions(*db)
	for _, tm := range tms {
		if tm.Selector < len(db.ActionTables.Transitions) {
			t := db.ActionTables.Transitions[tm.Selector]
			if t != nil {
				m := structs.Map(*t)
				m["coords"] = tm.Coords
				m["selector"] = tm.Selector
				m["location_name"] = defs.LocationString(t.Location)
				ms = append(ms, m)
			}
		}
	}

	return ms, nil
}

func dumpGame(bodies []msq.Body, dims []gen.Point) (map[string]interface{}, error) {
	var ms []map[string]interface{}

	for i, body := range bodies {
		blockMs, err := dumpBlock(body, dims[i])
		if err != nil {
			return nil, err
		}

		m := map[string]interface{}{
			"block":       i,
			"transitions": blockMs,
		}
		ms = append(ms, m)
	}

	return map[string]interface{}{
		"bodies": ms,
	}, nil
}

func dumpGames(bodies1 []msq.Body, bodies2 []msq.Body) (map[string]interface{}, error) {
	m1, err := dumpGame(bodies1, defs.MapDims[0])
	if err != nil {
		return nil, err
	}

	m2, err := dumpGame(bodies2, defs.MapDims[1])
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"game1": m1,
		"game2": m2,
	}, nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	log.SetLevel(log.InfoLevel)

	inDir := os.Args[1]

	descs0, descs1, err := wlutil.ReadAndParseGames(inDir)
	if err != nil {
		onErr(err)
	}

	bodies0 := wlutil.DescsToBodies(descs0)
	bodies1 := wlutil.DescsToBodies(descs1)

	m, err := dumpGames(bodies0[:defs.Block0NumBlocks], bodies1[:defs.Block1NumBlocks])
	if err != nil {
		onErr(err)
	}

	j, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		onErr(wlerr.Wrapf(err, "failed to marshal json"))
	}

	fmt.Printf("%s\n", j)
}
