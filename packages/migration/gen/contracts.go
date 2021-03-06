// Apla Software includes an integrated development
// environment with a multi-level system for the management
// of access rights to data, interfaces, and Smart contracts. The
// technical characteristics of the Apla Software are indicated in
// Apla Technical Paper.

// Apla Users are granted a permission to deal in the Apla
// Software without restrictions, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of Apla Software, and to permit persons
// to whom Apla Software is furnished to do so, subject to the
// following conditions:
// * the copyright notice of GenesisKernel and EGAAS S.A.
// and this permission notice shall be included in all copies or
// substantial portions of the software;
// * a result of the dealing in Apla Software cannot be
// implemented outside of the Apla Platform environment.

// THE APLA SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
// OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
// TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE, ERROR FREE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR
// THE USE OR OTHER DEALINGS IN THE APLA SOFTWARE.

package main

import (
	"bufio"
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	ext = ".sim"

	defaultPackageName = "migration"
)

var (
	scenarios = []scenario{
		{
			[]string{"./contracts/ecosystem"},
			"./contracts_data.go",
			"contractsDataSQL", "%[1]d", "%[2]d",
		},
		{
			[]string{"./contracts/common", "./contracts/first_ecosystem"},
			"./first_ecosys_contracts_data.go",
			"firstEcosystemContractsSQL", "1", "%[1]d",
		},
		{
			[]string{"./contracts/common", "./contracts/first_ecosystem", "./contracts/obs"},
			"./obs/obs_data_contracts.go",
			"contractsDataSQL", "%[1]d", "",
		},
	}

	propPrefix = []byte("// +prop ")
)

type scenario struct {
	Source    []string
	Dest      string
	Variable  string
	Ecosystem string
	Owner     string
}

type contract struct {
	Name       string
	Source     template.HTML
	Conditions template.HTML
	AppID      string
}

type meta struct {
	AppID      string
	Conditions string
}

var fns = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
}

var contractsTemplate = template.Must(template.New("").Funcs(fns).Parse(`// Code generated by go generate; DO NOT EDIT.

package {{ .Package }}

var {{ .Variable }} = ` + "`" + `
INSERT INTO "1_contracts" (id, name, value, conditions, app_id{{if .Owner }}, wallet_id{{end}}, ecosystem)
VALUES
{{- $last := add (len .Contracts) -1}}
{{- range $i, $item := .Contracts}}
	(next_id('1_contracts'), '{{ $item.Name }}', '{{ $item.Source }}', '{{ $item.Conditions }}', '{{ $item.AppID }}'{{if $.Owner }}, {{ $.Owner }}{{end}}, '{{ $.Ecosystem }}'){{if eq $last $i}};{{else}},{{end}}
{{- end}}
` + "`"))

func main() {
	for _, s := range scenarios {
		if err := generate(s); err != nil {
			panic(err)
		}
	}
}

func escape(data string) template.HTML {
	data = strings.Replace(data, `%`, `%%`, -1)
	data = strings.Replace(data, `'`, `''`, -1)
	data = strings.Replace(data, "`", "` + \"`\" + `", -1)
	return template.HTML(data)
}

func loadSource(srcPath string) (*contract, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	props := make([]byte, 0)
	data := make([]byte, 0)

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		line := scan.Bytes()
		if bytes.HasPrefix(line, propPrefix) {
			props = append(append(props, line[len(propPrefix):]...), '\n')
		} else {
			data = append(append(data, line...), '\n')
		}
	}

	m := &meta{}
	if err = toml.Unmarshal(props, m); err != nil {
		return nil, err
	}

	name := filepath.Base(srcPath)
	ext := filepath.Ext(srcPath)

	return &contract{
		Name:       name[0 : len(name)-len(ext)],
		Source:     escape(string(data)),
		Conditions: escape(m.Conditions),
		AppID:      m.AppID,
	}, nil
}

func loadSources(srcPaths []string) ([]*contract, error) {
	sources := make([]*contract, 0)

	for _, srcPath := range srcPaths {
		err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) != ext {
				return nil
			}

			source, err := loadSource(path)
			if err != nil {
				return err
			}

			sources = append(sources, source)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})

	return sources, nil
}

func generate(s scenario) error {
	sources, err := loadSources(s.Source)
	if err != nil {
		return err
	}

	file, err := os.Create(s.Dest)
	if err != nil {
		return err
	}
	defer file.Close()

	pkg := filepath.Base(filepath.Dir(s.Dest))
	if pkg == "." {
		pkg = defaultPackageName
	}

	return contractsTemplate.Execute(file, map[string]interface{}{
		"Package":   pkg,
		"Variable":  s.Variable,
		"Ecosystem": s.Ecosystem,
		"Owner":     nil,
		"Contracts": sources,
	})
}
