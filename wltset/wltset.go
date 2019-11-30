package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/badvassal/wllib/defs"
	"github.com/badvassal/wllib/gen/wlerr"
	"github.com/badvassal/wllib/wlutil"
	"github.com/badvassal/wlmanip"
)

var WltsetVersion = "0.0.1"

func onErr(err error) {
	fmt.Fprintf(os.Stderr, "* error: %s\n", err.Error())
	os.Exit(2)
}

type tsetCfg struct {
	Dir       string
	OpStrings []string
}

func parseOperand(s string) (*defs.LocPair, error) {
	s = strings.TrimSpace(s)

	var fromStr string
	var toStr string

	// First try to parse the format that gets logged.
	re := regexp.MustCompile(`\s*\[\d+\s+(\w+)\],\s*\[\d+\s+(\w+)\]`)
	submatches := re.FindStringSubmatch(s)
	if submatches != nil {
		fromStr = submatches[1]
		toStr = submatches[2]
	} else {
		parts := strings.Split(s, ",")
		if len(parts) != 2 {
			return nil, wlerr.Errorf(
				"invalid operand: wrong comma count: have=%d want=1 s=\"%s\"",
				len(parts), s)
		}

		fromStr = parts[0]
		toStr = parts[1]
	}

	from, err := wlmanip.ParseLocationNoCase(fromStr)
	if err != nil {
		return nil, err
	}

	to, err := wlmanip.ParseLocationNoCase(toStr)
	if err != nil {
		return nil, err
	}

	return &defs.LocPair{from, to}, nil
}

func parseTransOp(opString string) (*wlmanip.TransOp, error) {
	parts := strings.Split(opString, "<-")
	if len(parts) != 2 {
		return nil, wlerr.Errorf(
			"invalid operation: wrong `<-` count: "+
				"have=%d want=1 opString=\"%s\"",
			len(parts), opString)
	}

	dst, err := parseOperand(parts[0])
	if err != nil {
		return nil, err
	}

	src, err := parseOperand(parts[1])
	if err != nil {
		return nil, err
	}

	return &wlmanip.TransOp{
		A: *dst,
		B: *src,
	}, nil
}

func tset(cfg tsetCfg) error {
	var ops []wlmanip.TransOp
	for _, s := range cfg.OpStrings {
		op, err := parseTransOp(s)
		if err != nil {
			return err
		}

		ops = append(ops, *op)
	}

	blocks0, blocks1, err := wlutil.ReadAndParseGames(cfg.Dir)
	if err != nil {
		return err
	}

	state, err := wlutil.DecodeGames(blocks0, blocks1)
	if err != nil {
		return err
	}

	coll, err := wlmanip.Collect(*state, wlmanip.CollectCfg{})
	if err != nil {
		return err
	}

	for _, op := range ops {
		wlmanip.ExecTransOp(coll, state, op)
	}

	if err := wlutil.CommitDecodeState(*state, blocks0, blocks1); err != nil {
		return err
	}

	if err := wlutil.SerializeAndWriteGames(blocks0, blocks1, cfg.Dir); err != nil {
		return err
	}

	return nil
}

func main() {
	app := cli.NewApp()

	app.Name = "wltset"
	app.Usage = "Wasteland transition setter"
	app.ArgsUsage = "<op0> [op1] [...]"
	app.Version = WltsetVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:     "path,p",
			Usage:    "path of wasteland directory",
			Required: true,
		},

		cli.StringFlag{
			Name:  "loglevel,l",
			Usage: "log level; one of: debug, info, warn, error, panic",
			Value: "warn",
		},
	}

	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[options]{{end}}{{if .Commands}}{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`
	app.Action = func(c *cli.Context) error {
		lvl, err := log.ParseLevel(c.String("loglevel"))
		if err != nil {
			return wlerr.Errorf("invalid log level: \"%s\"", c.String("loglevel"))
		}
		log.SetLevel(lvl)

		if len(c.Args()) == 0 {
			return wlerr.Errorf("missing required `op0` argument")
		}

		return tset(tsetCfg{
			Dir:       c.String("path"),
			OpStrings: c.Args(),
		})
	}

	if err := app.Run(os.Args); err != nil {
		onErr(err)
	}
}
