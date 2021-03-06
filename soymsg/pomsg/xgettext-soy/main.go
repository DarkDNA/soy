// xgettext-soy is a tool to extract messages from Soy templates in the PO
// (gettext) file format.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DarkDNA/soy/ast"
	"github.com/DarkDNA/soy/parse"
	"github.com/DarkDNA/soy/parsepasses"
	"github.com/DarkDNA/soy/soymsg/pomsg"
	"github.com/DarkDNA/soy/template"
	"github.com/robfig/gettext/po"
)

func usage() {
	fmt.Fprintln(os.Stderr, `xgettext-soy is a tool to extract messages from Soy templates.

Usage:

	./xgettext-soy [INPUTPATH]...

INPUTPATH elements may be files or directories. Input directories will be
recursively searched for *.soy files.

The resulting POT (PO template) file is written to STDOUT
`)
}

var registry = template.Registry{}

func main() {
	if len(os.Args) < 2 || strings.HasSuffix(os.Args[1], "help") {
		usage()
		os.Exit(1)
	}

	// Add all the sources to the registry.
	for _, src := range os.Args[1:] {
		err := filepath.Walk(src, walkSource)
		if err != nil {
			exit(err)
		}
	}
	parsepasses.ProcessMessages(registry)

	var e = extractor{
		file:     &po.File{},
		registry: registry,
	}
	for _, t := range registry.Templates {
		e.tmpl = t.Node
		e.extract(t.Node)
	}
	e.file.WriteTo(os.Stdout)
}

func walkSource(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !strings.HasSuffix(path, ".soy") {
		return nil
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	tree, err := parse.SoyFile(path, string(content))
	if err != nil {
		return err
	}
	if err = registry.Add(tree); err != nil {
		return err
	}
	return nil
}

type extractor struct {
	file     *po.File
	tmpl     *ast.TemplateNode
	registry template.Registry
}

func (e extractor) extract(node ast.Node) {
	switch node := node.(type) {
	case *ast.MsgNode:
		if err := pomsg.Validate(node); err != nil {
			exit(err)
		}
		var pluralVar = ""
		if plural, ok := node.Body.Children()[0].(*ast.MsgPluralNode); ok {
			pluralVar = " var=" + plural.VarName
		}
		e.file.Messages = append(e.file.Messages, po.Message{
			Comment: po.Comment{
				ExtractedComments: []string{node.Desc},
				References: []string{
					fmt.Sprintf("%s:%d",
						e.registry.SourceFile(e.tmpl.Name),
						e.registry.LineNumber(e.tmpl.Name, node)),
					fmt.Sprintf("id=%d%v", node.ID, pluralVar)},
			},
			Ctxt:     node.Meaning,
			Id:       pomsg.Msgid(node),
			IdPlural: pomsg.MsgidPlural(node),
			Str: []string{
				pomsg.Msgid(node),
				pomsg.MsgidPlural(node),
			},
		})
	case ast.ParentNode:
		for _, child := range node.Children() {
			e.extract(child)
		}
	default:
		break
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
