package generate

import (
	"github.com/go-swagger/go-swagger/generator"
	"github.com/jessevdk/go-flags"
)

// Antd generates a markdown representation of the spec
type Antd struct {
	WithShared
	WithModels
	WithOperations

	Output flags.Filename `long:"output" short:"" description:"the file to write the generated markdown." default:"markdown.md"`
}

func (m Antd) apply(opts *generator.GenOpts) {
	m.Shared.apply(opts)
	m.Models.apply(opts)
	m.Operations.apply(opts)

	opts.IncludeModel = true
	opts.IncludeHandler = true

}

func (m *Antd) generate(opts *generator.GenOpts) error {
	return generator.GenerateAntd(string(m.Output), m.Models.Models, m.Operations.Operations, opts)
}

func (m Antd) log(rp string) {
}

// Execute runs this command
func (m *Antd) Execute(args []string) error {
	return createSwagger(m)
}
