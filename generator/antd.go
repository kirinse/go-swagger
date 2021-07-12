package generator

import (
	"errors"
	"github.com/go-openapi/inflect"
	"github.com/go-openapi/swag"
	"strings"
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

	// data.ts
	//  Target:   "{{ joinFilePath .Target (trimPrefix .Name "antd_model:") }}",
	//	FileName: "data.ts",
	opgImports := make(map[string]map[string]string)
	modelMap := make(map[string]*GenDefinition)
	for _, m := range app.Models {
		if m.IsStream {
			continue
		}
		mod := m
		modelName := strings.TrimPrefix(mod.Description, "antd_model:")
		if _, ok := opgImports[modelName]; !ok {
			opgImports[modelName] = make(map[string]string)
		}
		opgImports[modelName][strings.Replace(mod.Name, strings.ToLower(app.Name), "", 1)] = ""
		// .CustomTag 用于替换前缀
		mod.GenSchema.CustomTag = app.Name
		// .Package 用于确定目录
		mod.Package = modelName
		mod.Suffix = modelName
		if _, ok := modelMap[modelName]; !ok {
			modelMap[modelName] = &mod
		}

		modelMap[modelName].ExtraSchemas = append(modelMap[modelName].ExtraSchemas, mod.GenSchema)
	}
	for _, mod := range modelMap {
		if err := c.GenOpts.renderDefinition(mod); err != nil {
			return err
		}
	}

	// _list.tsx
	//  Target:   "{{ joinFilePath .Target (pascalize .Name) }}",
	//	FileName: "_list.tsx",
	// service.ts
	//  Target:   "{{ joinFilePath .Target (pascalize .Name) }}",
	//	FileName: "service.ts",
	debugLog("---- app.Name: %s\n", app.Name)
	debugLog("---- total app.OperationGroups: %d\n", len(app.OperationGroups))
	for _, opg := range app.OperationGroups {
		opg.RootPackage = app.Name
		pascalizedName := pascalize(opg.Name)
		debugLog("---- app.OperationGroup.Name: %s -> %+v\n", opg.Name, pascalizedName)
		debugLog("---- OperationGroup.RootPackage: %s -> %+v\n", opg.RootPackage, pascalize(opg.RootPackage))
		if imports, ok := opgImports[pascalizedName]; ok {
			opg.Imports = imports
		}
		for _, op := range opg.Operations {
			op.Package = pascalize(opg.Name)
			op.RootPackage = app.Name
			expectedMethod := "List"+inflect.Pluralize(op.Package)
			debugLog(
				"---- operation Name: %s, Package: %s, RootPackage: %s, want: %s\n",
				op.Name,
				op.Package,
				op.RootPackage,
				expectedMethod,
			)
			if op.Name == expectedMethod {
				if err := c.GenOpts.renderOperation(&op); err != nil {
					return err
				}
			}
		}
		if err := c.GenOpts.renderOperationGroup(&opg); err != nil {
			return err
		}
	}
	return nil
}

// AntdOpts for rendering a spec as markdown
func AntdOpts() *LanguageOpts {
	opts := &LanguageOpts{
		fileNameFunc: nil, // func(string) string // language specific source file naming rules
		dirNameFunc:  nil, // func(string) string // language specific directory naming rules
	}
	opts.Init()
	return opts
}

// AntdSectionOpts for a given opts and output file.
func AntdSectionOpts(gen *GenOpts, output string) {
	gen.Sections.Models = []TemplateOpts{
		{
			Name:     "antd data",
			Source:   "asset:antdDatag",
			Target:   "{{ joinFilePath .Target .Package }}",
			FileName: "data.ts",
		},
		{
			Name:     "antd columns",
			Source:   "asset:antdColumns",
			Target:   "{{ joinFilePath .Target .Package }}",
			FileName: "columns.tsx",
		},
	}
	gen.Sections.OperationGroups = []TemplateOpts{
		{
			Name:     "antd service",
			Source:   "asset:antdOpg",
			Target:   "{{ joinFilePath .Target (pascalize .Name) }}",
			FileName: "service.ts",
		},
	}
	gen.Sections.Operations = []TemplateOpts{
		{
			Name:     "antd list",
			Source:   "asset:antdList",
			Target:   "{{ joinFilePath .Target .Package }}",
			FileName: "list.tsx",
		},
	}
	gen.LanguageOpts = AntdOpts()
	gen.Sections.Application = []TemplateOpts{}
}
