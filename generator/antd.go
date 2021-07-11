package generator

import (
	"errors"
	"github.com/go-openapi/swag"
	"path/filepath"
)

// GenerateAntd documentation for a swagger specification
func GenerateAntd(output string, modelNames, operationIDs []string, opts *GenOpts) error {
	if err := opts.CheckOpts(); err != nil {
		return err
	}

	if err := opts.setTemplates(); err != nil {
		return err
	}

	specDoc, analyzed, err := opts.analyzeSpec()
	if err != nil {
		return err
	}

	models, err := gatherModels(specDoc, modelNames)
	if err != nil {
		return err
	}

	operations := gatherOperations(analyzed, operationIDs)
	if len(operations) == 0 {
		return errors.New("no operations were selected")
	}

	generator := appGenerator{
		Name:              appNameOrDefault(specDoc, "", defaultClientName),
		SpecDoc:           specDoc,
		Analyzed:          analyzed,
		Models:            models,
		Operations:        operations,
		Target:            opts.Target,
		DumpData:          opts.DumpData,
		Package:           opts.LanguageOpts.ManglePackageName(opts.ClientPackage, defaultClientTarget),
		APIPackage:        opts.LanguageOpts.ManglePackagePath(opts.APIPackage, defaultOperationsTarget),
		ModelsPackage:     opts.LanguageOpts.ManglePackagePath(opts.ModelPackage, defaultModelsTarget),
		ServerPackage:     opts.LanguageOpts.ManglePackagePath(opts.ServerPackage, defaultServerTarget),
		ClientPackage:     opts.LanguageOpts.ManglePackagePath(opts.ClientPackage, defaultClientTarget),
		OperationsPackage: opts.LanguageOpts.ManglePackagePath(opts.ClientPackage, defaultClientTarget),
		Principal:         opts.PrincipalAlias(),
		DefaultScheme:     opts.DefaultScheme,
		DefaultProduces:   opts.DefaultProduces,
		DefaultConsumes:   opts.DefaultConsumes,
		GenOpts:           opts,
	}
	return (&antdGenerator{generator}).Generate()


	//output = filepath.Join(opts.Target, output)
	//if err := opts.EnsureDefaults(); err != nil {
	//	return err
	//}
	//AntdSectionOpts(opts, output)
	//
	//generator, err := newAppGenerator("", modelNames, operationIDs, opts)
	//if err != nil {
	//	return err
	//}
	//
	//return generator.GenerateAntd()
}

type antdGenerator struct {
	appGenerator
}

func (c *antdGenerator) GenerateClient() error {
	app, err := c.makeCodegenApp()
	if err != nil {
		return err
	}

	if c.DumpData {
		return dumpData(swag.ToDynamicJSON(app))
	}

	if c.GenOpts.IncludeModel {
		for _, m := range app.Models {
			if m.IsStream {
				continue
			}
			mod := m
			if err := c.GenOpts.renderDefinition(&mod); err != nil {
				return err
			}
		}
	}

	if c.GenOpts.IncludeHandler {
		for _, g := range app.OperationGroups {
			opg := g
			for _, o := range opg.Operations {
				op := o
				if err := c.GenOpts.renderOperation(&op); err != nil {
					return err
				}
			}
			if err := c.GenOpts.renderOperationGroup(&opg); err != nil {
				return err
			}
		}
	}

	if c.GenOpts.IncludeSupport {
		if err := c.GenOpts.renderApplication(&app); err != nil {
			return err
		}
	}

	return nil
}

func (c *antdGenerator) Generate() error {
	app, err := c.makeCodegenApp()
	if err != nil {
		return err
	}
	//output := filepath.Join(c.Target, output)
	if err := c.GenOpts.EnsureDefaults(); err != nil {
		return err
	}
	AntdSectionOpts(c.GenOpts, c.Target)

	//generator, err := newAppGenerator("", modelNames, operationIDs, opts)
	//if err != nil {
	//	return err
	//}


	return c.GenOpts.renderApplication(&app)
}


// AntdOpts for rendering a spec as markdown
func AntdOpts() *LanguageOpts {
	opts := &LanguageOpts{}
	opts.Init()
	return opts
}

// AntdSectionOpts for a given opts and output file.
func AntdSectionOpts(gen *GenOpts, output string) {
	gen.Sections.Models = nil
	gen.Sections.OperationGroups = nil
	gen.Sections.Operations = nil
	gen.LanguageOpts = AntdOpts()
	gen.Sections.Application = []TemplateOpts{
		{
			Name:     "antdData",
			Source:   "asset:antdData",
			Target:   filepath.Dir(output),
			FileName: "data.ts",
		},
		{
			Name:     "antdColumns",
			Source:   "asset:antdColumns",
			Target:   filepath.Dir(output),
			FileName: "columns.tsx",
		},
		{
			Name:     "antdList",
			Source:   "asset:antdList",
			Target:   filepath.Dir(output),
			FileName: "_list.tsx",
		},
		{
			Name:     "antdService",
			Source:   "asset:antdService",
			Target:   filepath.Dir(output),
			FileName: "service.ts",
		},
	}
}
