// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/googleapis/librarian/internal/yaml"
)

// flowSpec is the YAML body of a ```flow block: a swimlane diagram with one
// row per lane and one column per node, rendered as inline SVG so pages need
// no JavaScript or network access to show it.
type flowSpec struct {
	Title string     `yaml:"title"`
	Lanes []flowLane `yaml:"lanes"`
	Nodes []flowNode `yaml:"nodes"`
	// Edges are written as "from -> to" with an optional " | label".
	Edges []string `yaml:"edges"`
}

type flowLane struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
	Kind  string `yaml:"kind"`
}

type flowNode struct {
	ID    string `yaml:"id"`
	Lane  string `yaml:"lane"`
	Label string `yaml:"label"`
	Kind  string `yaml:"kind"`
	Step  int    `yaml:"step"`
	Col   *int   `yaml:"col"`
	col   int
	row   int
}

const (
	laneLabelW = 124
	nodeW      = 144
	nodeH      = 52
	colGap     = 32
	laneH      = 84
	padX       = 14
	padY       = 10
	charsLine  = 20
	// backEdgeRise is the headroom reserved above the top lane when an edge
	// loops back to an earlier column.
	backEdgeRise = 36
)

func renderFlow(body string) (string, error) {
	spec, err := yaml.Unmarshal[flowSpec]([]byte(body))
	if err != nil {
		return "", fmt.Errorf("flow: %w", err)
	}
	if len(spec.Lanes) == 0 || len(spec.Nodes) == 0 {
		return "", fmt.Errorf("flow: lanes and nodes are required")
	}
	lanes := map[string]int{}
	for i, l := range spec.Lanes {
		lanes[l.ID] = i
	}
	nodes := map[string]*flowNode{}
	next, cols := 0, 0
	for i := range spec.Nodes {
		n := &spec.Nodes[i]
		row, ok := lanes[n.Lane]
		if !ok {
			return "", fmt.Errorf("flow: node %q uses unknown lane %q", n.ID, n.Lane)
		}
		n.row = row
		if n.Col != nil {
			n.col = *n.Col
		} else {
			n.col = next
		}
		next = n.col + 1
		cols = max(cols, n.col+1)
		if n.Kind == "" {
			n.Kind = spec.Lanes[row].Kind
		}
		nodes[n.ID] = n
	}
	type edge struct {
		from, to *flowNode
		label    string
	}
	var edges []edge
	top := 0
	for _, e := range spec.Edges {
		from, to, label, err := parseEdge(e)
		if err != nil {
			return "", err
		}
		a, c := nodes[from], nodes[to]
		if a == nil || c == nil {
			return "", fmt.Errorf("flow: edge %q references an unknown node", e)
		}
		if c.col < a.col && min(a.row, c.row) == 0 {
			top = backEdgeRise
		}
		edges = append(edges, edge{a, c, label})
	}
	width := laneLabelW + padX*2 + cols*nodeW + (cols-1)*colGap
	height := top + padY*2 + len(spec.Lanes)*laneH
	var b strings.Builder
	b.WriteString(`<figure class="flow">`)
	if spec.Title != "" {
		b.WriteString(`<figcaption>` + html.EscapeString(spec.Title) + `</figcaption>`)
	}
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" width="%d" role="img" xmlns="http://www.w3.org/2000/svg">`, width, height, width)
	b.WriteString(`<defs><marker id="wt-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="arrowhead"/></marker></defs>`)
	fmt.Fprintf(&b, `<g transform="translate(0 %d)">`, top)
	for i, l := range spec.Lanes {
		y := padY + i*laneH
		fmt.Fprintf(&b, `<rect class="lane lane-%s" x="0" y="%d" width="%d" height="%d"/>`, html.EscapeString(kindOr(l.Kind)), y, width, laneH)
		fmt.Fprintf(&b, `<text class="lane-label" x="%d" y="%d">`, padX, y+laneH/2)
		writeLines(&b, wrap(l.Label, 20), padX, y+laneH/2)
		b.WriteString(`</text>`)
	}
	for _, e := range edges {
		writeEdge(&b, e.from, e.to, e.label)
	}
	for i := range spec.Nodes {
		n := &spec.Nodes[i]
		x, y := nodeX(n.col), nodeY(n.row)
		if n.Step > 0 {
			fmt.Fprintf(&b, `<a href="#step-%d" class="node-link">`, n.Step)
		}
		fmt.Fprintf(&b, `<g class="node kind-%s"><rect x="%d" y="%d" width="%d" height="%d" rx="8"/>`, html.EscapeString(kindOr(n.Kind)), x, y, nodeW, nodeH)
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle">`, x+nodeW/2, y+nodeH/2)
		writeLines(&b, wrap(n.Label, charsLine), x+nodeW/2, y+nodeH/2)
		b.WriteString(`</text></g>`)
		if n.Step > 0 {
			b.WriteString(`</a>`)
		}
	}
	b.WriteString(`</g></svg></figure>`)
	return b.String(), nil
}

func kindOr(kind string) string {
	if kind == "" {
		return "default"
	}
	return kind
}

func nodeX(col int) int { return laneLabelW + padX + col*(nodeW+colGap) }
func nodeY(row int) int { return padY + row*laneH + (laneH-nodeH)/2 }

func parseEdge(e string) (from, to, label string, err error) {
	spec, label, _ := strings.Cut(e, "|")
	from, to, ok := strings.Cut(spec, "->")
	if !ok {
		return "", "", "", fmt.Errorf("flow: edge %q must look like \"from -> to | label\"", e)
	}
	return strings.TrimSpace(from), strings.TrimSpace(to), strings.TrimSpace(label), nil
}

func writeEdge(b *strings.Builder, a, c *flowNode, label string) {
	var d string
	var lx, ly int
	switch {
	case c.col > a.col:
		x1, y1 := nodeX(a.col)+nodeW, nodeY(a.row)+nodeH/2
		x2, y2 := nodeX(c.col), nodeY(c.row)+nodeH/2
		d = fmt.Sprintf("M %d %d C %d %d, %d %d, %d %d", x1, y1, x1+colGap/2, y1, x2-colGap/2, y2, x2, y2)
		lx, ly = (x1+x2)/2, (y1+y2)/2-6
	case c.col == a.col:
		x := nodeX(a.col) + nodeW/2
		y1, y2 := nodeY(a.row)+nodeH, nodeY(c.row)
		if c.row < a.row {
			y1, y2 = nodeY(a.row), nodeY(c.row)+nodeH
		}
		d = fmt.Sprintf("M %d %d L %d %d", x, y1, x, y2)
		lx, ly = x+8, (y1+y2)/2
	default:
		x1, y1 := nodeX(a.col)+nodeW/2, nodeY(a.row)
		x2, y2 := nodeX(c.col)+nodeW/2, nodeY(c.row)
		d = fmt.Sprintf("M %d %d C %d %d, %d %d, %d %d", x1, y1, x1, y1-34, x2, y2-34, x2, y2)
		lx, ly = (x1+x2)/2, min(y1, y2)-28
	}
	fmt.Fprintf(b, `<path class="edge" d="%s" marker-end="url(#wt-arrow)"/>`, d)
	if label != "" {
		w := len(label)*6 + 10
		fmt.Fprintf(b, `<rect class="edge-label-bg" x="%d" y="%d" width="%d" height="16" rx="3"/>`, lx-w/2, ly-12, w)
		fmt.Fprintf(b, `<text class="edge-label" x="%d" y="%d" text-anchor="middle">%s</text>`, lx, ly, html.EscapeString(label))
	}
}

// wrap breaks text into lines of at most width characters at word
// boundaries.
func wrap(text string, width int) []string {
	var lines []string
	var cur string
	for w := range strings.FieldsSeq(text) {
		switch {
		case cur == "":
			cur = w
		case len(cur)+1+len(w) <= width:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// writeLines emits tspans vertically centered on y.
func writeLines(b *strings.Builder, lines []string, x, y int) {
	const lineH = 14
	top := y - (len(lines)-1)*lineH/2 + 4
	for i, l := range lines {
		fmt.Fprintf(b, `<tspan x="%d" y="%d">%s</tspan>`, x, top+i*lineH, html.EscapeString(l))
	}
}
