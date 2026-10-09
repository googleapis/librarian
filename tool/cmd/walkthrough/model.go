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

// SiteData is the complete JSON payload embedded in the generated HTML site.
// Agents and local tools can inspect or append to Guides to add custom tours.
type SiteData struct {
	SHA              string                 `json:"sha"`
	ShortSHA         string                 `json:"shortSha"`
	GitRef           string                 `json:"gitRef"`
	Repo             string                 `json:"repo"`
	GoVersion        string                 `json:"goVersion"`
	LibrarianVersion string                 `json:"librarianVersion"`
	TotalLOC         int                    `json:"totalLoc"`
	MaxEmbeddedLines int                    `json:"maxEmbeddedLines"`
	Layers           []Layer                `json:"layers"`
	Pkgs             []Pkg                  `json:"pkgs"`
	Steps            []TourStep             `json:"steps"`
	Flows            []CommandFlow          `json:"flows"`
	Langs            []LangInfo             `json:"langs"`
	ContractCols     []string               `json:"contractCols"`
	ContractHooks    []ContractRow          `json:"contractHooks"`
	Findings         []Finding              `json:"findings"`
	Guides           []Guide                `json:"guides"`
	Files            map[string]SourceEntry `json:"files"`
}

// SourceEntry holds live repository file or directory preview content embedded
// into the standalone HTML artifact so source links inspect code directly in-page.
type SourceEntry struct {
	IsDir     bool     `json:"dir,omitempty"`
	Lines     int      `json:"lines,omitempty"`
	Truncated bool     `json:"trunc,omitempty"`
	Pkg       string   `json:"pkg,omitempty"`
	Entries   []string `json:"entries,omitempty"`
	Content   string   `json:"content,omitempty"`
}

// Layer defines one architectural tier in the interactive dependency map.
type Layer struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Hint  string `json:"hint"`
}

// Pkg describes a Go package (or grouped directory such as tool/cmd) in the map.
type Pkg struct {
	ID      string   `json:"id"`
	Layer   string   `json:"layer"`
	Side    string   `json:"side,omitempty"`
	LOC     int      `json:"loc"`
	Path    string   `json:"path,omitempty"`
	Desc    string   `json:"desc"`
	Imports []string `json:"imports"`
}

// TourStep is a step in the main Guided Architecture Tour.
type TourStep struct {
	Title string   `json:"t"`
	Body  string   `json:"body"`
	Look  []string `json:"look"`
	Pkg   string   `json:"pkg,omitempty"`
}

// FlowStage represents one stage node in a visual CommandFlow pipeline diagram.
type FlowStage struct {
	Title string `json:"title"`
	Where string `json:"where"`
	Body  string `json:"body"`
	Out   string `json:"out,omitempty"`
}

// CommandFlow describes a CLI command execution pipeline with explicit inputs and stages.
type CommandFlow struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Cmd    string      `json:"cmd"`
	File   string      `json:"file"`
	Inputs []string    `json:"inputs"`
	Stages []FlowStage `json:"stages"`
}

// LangInfo summarizes one of the nine language integrations under internal/librarian.
type LangInfo struct {
	Name   string `json:"n"`
	Pkg    string `json:"pkg"`
	LOC    int    `json:"loc"`
	Gen    string `json:"gen"`
	Inst   string `json:"inst"`
	Fmt    string `json:"fmt"`
	Meta   bool   `json:"meta"`
	Bump   bool   `json:"bump"`
	Pub    string `json:"pub"`
	CI     string `json:"ci"`
	Detail string `json:"detail"`
}

// ContractRow records which exported hooks each language package implements.
type ContractRow struct {
	Hook   string   `json:"hook"`
	Caller string   `json:"caller"`
	Cells  []string `json:"cells"`
}

// Finding highlights a live codebase observation or architectural quirk.
type Finding struct {
	Title string `json:"t"`
	Body  string `json:"b"`
	File  string `json:"f"`
	Pkg   string `json:"pkg,omitempty"`
}

// Guide is a multi-step interactive deep dive selectable from the landing page or Guides tab.
// Custom guides dropped into .agents/skills/walkthrough/custom/*.json use this schema.
type Guide struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Subtitle string      `json:"subtitle"`
	Audience string      `json:"audience"`
	Custom   bool        `json:"custom,omitempty"`
	Steps    []GuideStep `json:"steps"`
}

// GuideStep is a single step within a Guide.
type GuideStep struct {
	Title string   `json:"t"`
	Runs  string   `json:"runs,omitempty"`
	Code  string   `json:"code,omitempty"`
	Body  string   `json:"body"`
	Look  []string `json:"look,omitempty"`
	Pkg   string   `json:"pkg,omitempty"`
}
