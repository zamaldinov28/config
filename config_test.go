package config

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

func TestNewParser(t *testing.T) {
	type testStruct struct {
		Help              bool   `config:"name:help;mode:cli;default:f;desc:Lorem ipsum"`
		ConfigFile        string `config:"name:config_file;mode:cli"`
		Prefix            string `config:"name:prefix;mode:cli;default:;desc:"`
		Ignored           string
		alsoIgnored       string
		alsoShouldIgnored string `some:"another_tag"`
		Nested            struct {
			Int       int `config:"name:int"`
			NestedTwo struct {
				Bool   bool   `config:"name:nestedtwo.bool"`
				String string `config:"name:string"`
				Ignore string
			} `config:"mode:cli"`
		} `config:"name:nested;mode:cli,env"`
	}
	type errTestStruct struct {
		Help bool `config:"name:help;mode:ZZZ;default:f;desc:Lorem ipsum"`
	}
	type errNestedModeStruct struct {
		Nested struct {
			Int    int `config:"name:int"`
			Nested struct {
				Bool bool `config:"name:bool"`
			} `config:"name:nested;mode:cfg"` // Should be cli or/and env
		} `config:"name:nested;mode:cli,env"`
	}
	type args struct {
		in interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    Parser
		wantErr bool
	}{
		{name: "struct", args: args{in: testStruct{}}, want: Parser{}, wantErr: true},
		{name: "pointer", args: args{in: &testStruct{}}, want: Parser{in: &testStruct{}, fields: map[string]*structField{
			"Help":                    {name: "Help", tags: structFieldTags{name: "help", mode: modeCli, defaultValue: "f", hasDefaultValue: true, description: "Lorem ipsum", hasDescription: true}},
			"ConfigFile":              {name: "ConfigFile", tags: structFieldTags{name: "config_file", mode: modeCli}},
			"Prefix":                  {name: "Prefix", tags: structFieldTags{name: "prefix", mode: modeCli, defaultValue: "", hasDefaultValue: true, description: "", hasDescription: true}},
			"Nested.Int":              {name: "Nested.Int", tags: structFieldTags{name: "nested.int", mode: modeCli | modeEnv}},
			"Nested.NestedTwo.Bool":   {name: "Nested.NestedTwo.Bool", tags: structFieldTags{name: "nested.nestedtwo.bool", mode: modeCli}},
			"Nested.NestedTwo.String": {name: "Nested.NestedTwo.String", tags: structFieldTags{name: "nested.string", mode: modeCli}},
		}}, wantErr: false},
		{name: "err", args: args{in: &errTestStruct{}}, wantErr: true},
		{name: "err nested mode", args: args{in: &errNestedModeStruct{}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewParser(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_Help(t *testing.T) {
	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		prefix string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "blank",
			fields: fields{
				fields: map[string]*structField{
					"first_field": {
						name: "short_field",
						tags: structFieldTags{
							name:            "b",
							defaultValue:    "1",
							hasDefaultValue: true,
							description:     "Some description",
							hasDescription:  true,
						},
					},
					"second_field": {
						name: "long_field",
						tags: structFieldTags{
							name:           "afffffff",
							mode:           modeCli | modeCfg,
							description:    "Some more description",
							hasDescription: true,
						},
					},
					"third_field": {
						name: "long_field",
						tags: structFieldTags{
							name:           "cfffffffff",
							mode:           modeCli | modeCfg | modeEnv,
							description:    "Some more more description",
							hasDescription: true,
						},
					},
					"fourth_field": {
						name: "field_with_no_desc",
						tags: structFieldTags{
							name: "cxxxxxxxx",
							mode: modeCli | modeCfg | modeEnv,
						},
					},
					"fifth_field": {
						name: "field_with_empty_desc",
						tags: structFieldTags{
							name:           "yyyyyyyy",
							mode:           modeCli,
							description:    "",
							hasDescription: true,
						},
					},
					"nested_field": {
						name: "nested_field",
						tags: structFieldTags{
							name:           "nested.field",
							mode:           modeCli | modeCfg,
							description:    "Nested field example",
							hasDescription: true,
						},
					},
				},
			},
			want: `--afffffff     Some more description (cli, cfg only)
--b[=1]        Some description
--cfffffffff   Some more more description
--nested.field Nested field example (cli, cfg only)
--yyyyyyyy     (cli only)
`,
		},
		{
			name: "prefix with sort check",
			fields: fields{
				fields: map[string]*structField{
					"first_field": {
						name: "short_field",
						tags: structFieldTags{
							name:            "f",
							defaultValue:    "1",
							hasDefaultValue: true,
							description:     "Some description",
							hasDescription:  true,
						},
					},
					"second_field": {
						name: "short_field",
						tags: structFieldTags{
							name:            "ff",
							defaultValue:    "2",
							hasDefaultValue: true,
							description:     "Some description two",
							hasDescription:  true,
						},
					},
				},
			},
			args: args{prefix: "        "},
			want: `        --f[=1]  Some description
        --ff[=2] Some description two
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			if got := p.Help(tt.args.prefix); got != tt.want {
				t.Errorf("Parser.Help() = \n%v\n, want \n%v\n", got, tt.want)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	type errTestStructFile struct {
		Help       bool   `config:"name:help;mode:cli;default:f;desc:Lorem ipsum"`
		ConfigFile string `config:"name:config_file;mode:cli;desc:Lorem ipsum"`
		Prefix     string `config:"name:prefix;mode:cli;default:;desc:Lorem ipsum"`
	}
	type errTestStructConv struct {
		West int `config:"name:best;mode:env;default:ss;desc:best"`
	}
	type goodStruct struct {
		ConfigFile string `config:"name:good_config_file;mode:cli;desc:Lorem ipsum"`
		Test       string `config:"name:test;mode:env;desc:test"`
		Prefix     int    `config:"name:prefix;mode:cli;default:50;desc:best"`
		Ignore     string
		Nested     struct {
			Int       int `config:"name:int"`
			NestedTwo struct {
				Bool   bool   `config:"name:nestedtwo.bool"`
				String string `config:"name:string"`
				Ignore string
			} `config:""`
		} `config:"name:nested"`
	}
	type defaultValuesStruct struct {
		ConfigFile string `config:"name:config_file_not_from_cli;mode:cli;default:/will/be/replaced/later.json;desc:Lorem ipsum"`
		Prefix     string `config:"name:prefix_not_from_cli;mode:cli;default:TEST_;desc:Lorem ipsum"`
	}

	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		cfgPathConfig   string
		envPrefixConfig string
	}

	t.Setenv("100_TEST", "100")

	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "config_*.json")
	if err != nil {
		t.Error(err)
	}

	err = f.Chmod(0777)
	if err != nil {
		t.Error(err)
	}

	_, err = f.WriteString(`{"prefix":"100}`) // With error
	if err != nil {
		t.Error(err)
	}

	fgood, err := os.CreateTemp(dir, "config_*.json")
	if err != nil {
		t.Error(err)
	}

	err = fgood.Chmod(0777)
	if err != nil {
		t.Error(err)
	}

	_, err = fgood.WriteString(`{"test":"100","nested":{"int":5,"string":"lorem","nestedtwo":{"bool":true}}}`)
	if err != nil {
		t.Error(err)
	}

	os.Args = []string{"/app/test", "zzz", fmt.Sprintf("--config_file=%s", f.Name()), fmt.Sprintf("--good_config_file=%s", fgood.Name()), "--prefix=100"}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "broken file",
			fields: fields{in: &errTestStructFile{}, fields: map[string]*structField{
				"Help":       {name: "Help", tags: structFieldTags{name: "help", mode: modeCli, defaultValue: "f", hasDefaultValue: true}},
				"ConfigFile": {name: "ConfigFile", tags: structFieldTags{name: "config_file", mode: modeCli}},
				"Prefix":     {name: "Prefix", tags: structFieldTags{name: "prefix", mode: modeCli, defaultValue: "", hasDefaultValue: true}},
			}},
			args:    args{cfgPathConfig: "config_file", envPrefixConfig: "prefix"},
			wantErr: true,
		},
		{
			name: "error conv",
			fields: fields{in: &errTestStructConv{}, fields: map[string]*structField{
				"West": {name: "West", tags: structFieldTags{name: "best", mode: modeEnv, defaultValue: "ss", hasDefaultValue: true, description: "best"}},
			}},
			args:    args{cfgPathConfig: "config_file", envPrefixConfig: "prefix"},
			wantErr: true,
		},
		{
			name: "good struct",
			fields: fields{in: &goodStruct{}, fields: map[string]*structField{
				"ConfigFile":              {name: "ConfigFile", tags: structFieldTags{name: "good_config_file", mode: modeCli}},
				"Test":                    {name: "Test", tags: structFieldTags{name: "test", mode: modeEnv, description: "test"}},
				"Prefix":                  {name: "Prefix", tags: structFieldTags{name: "prefix", mode: modeCli, defaultValue: "50", hasDefaultValue: true, description: "best"}},
				"Nested.Int":              {name: "Nested.Int", tags: structFieldTags{name: "nested.int"}},
				"Nested.NestedTwo.Bool":   {name: "Nested.NestedTwo.Bool", tags: structFieldTags{name: "nested.nestedtwo.bool"}},
				"Nested.NestedTwo.String": {name: "Nested.NestedTwo.String", tags: structFieldTags{name: "nested.string"}},
			}},
			args:    args{cfgPathConfig: "good_config_file", envPrefixConfig: "prefix"},
			wantErr: false,
		},
		{
			name: "default values struct",
			fields: fields{in: &defaultValuesStruct{}, fields: map[string]*structField{
				"ConfigFile": {name: "ConfigFile", tags: structFieldTags{name: "config_file_not_from_cli", mode: modeCli, defaultValue: fgood.Name(), hasDefaultValue: true}},
				"Prefix":     {name: "Prefix", tags: structFieldTags{name: "prefix_not_from_cli", mode: modeCli, defaultValue: "TEST_", hasDefaultValue: true}},
			}},
			args:    args{cfgPathConfig: "config_file_not_from_cli", envPrefixConfig: "prefix_not_from_cli"},
			wantErr: false,
		},
		{
			name: "default values struct broken",
			fields: fields{in: &defaultValuesStruct{}, fields: map[string]*structField{
				"ConfigFile": {name: "ConfigFile", tags: structFieldTags{name: "config_file_not_from_cli", mode: modeCli, defaultValue: f.Name(), hasDefaultValue: true}},
				"Prefix":     {name: "Prefix", tags: structFieldTags{name: "prefix_not_from_cli", mode: modeCli, defaultValue: "TEST_", hasDefaultValue: true}},
			}},
			args:    args{cfgPathConfig: "config_file_not_from_cli", envPrefixConfig: "prefix_not_from_cli"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			if err := p.Parse(tt.args.cfgPathConfig, tt.args.envPrefixConfig); (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_fillStructWithValues(t *testing.T) {
	type goodStruct struct {
		String    string `config:"name:string"`
		StringTwo string `config:"name:string_two"`
		Nested    struct {
			Int       int `config:"name:int"`
			NestedTwo struct {
				Bool   bool   `config:"name:nestedtwo.bool"`
				String string `config:"name:string"`
				Ignore string
			} `config:""`
		} `config:"name:nested"`
	}

	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		target interface{}
		prefix string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "good",
			fields: fields{
				in: &goodStruct{},
				fields: map[string]*structField{
					"String":                  {name: "String", tags: structFieldTags{name: "string", defaultValue: "test", hasDefaultValue: true}},
					"StringTwo":               {name: "StringTwo", tags: structFieldTags{name: "string"}},
					"Nested.Int":              {name: "Nested.Int", tags: structFieldTags{name: "nested.int"}},
					"Nested.NestedTwo.Bool":   {name: "Nested.NestedTwo.Bool", tags: structFieldTags{name: "nested.nestedtwo.bool"}},
					"Nested.NestedTwo.String": {name: "Nested.NestedTwo.String", tags: structFieldTags{name: "nested.string"}},
				},
				parsedCfg: map[string]string{
					"nested.int":            "321",
					"nested.nestedtwo.bool": "true",
					"nested.string":         "qwerty",
				},
			},
			args: args{target: &goodStruct{}, prefix: ""},
		},
		{
			name: "error",
			fields: fields{
				in: &goodStruct{},
				fields: map[string]*structField{
					"String":                  {name: "String", tags: structFieldTags{name: "string"}},
					"Nested.Int":              {name: "Nested.Int", tags: structFieldTags{name: "nested.int"}},
					"Nested.NestedTwo.Bool":   {name: "Nested.NestedTwo.Bool", tags: structFieldTags{name: "nested.nestedtwo.bool"}},
					"Nested.NestedTwo.String": {name: "Nested.NestedTwo.String", tags: structFieldTags{name: "nested.string"}},
				},
				parsedCfg: map[string]string{
					"string":                "lorem",
					"nested.int":            "ZZZ",
					"nested.nestedtwo.bool": "true",
					"nested.string":         "qwerty",
				},
			},
			args:    args{target: &goodStruct{}, prefix: ""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			if err := p.fillStructWithValues(tt.args.target, tt.args.prefix); (err != nil) != tt.wantErr {
				t.Errorf("Parser.fillStructWithValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_newStructField(t *testing.T) {
	type str struct {
		ConfigFile string `config:"name:config_file;mode:cli;desc:Lorem ipsum"`
		Prefix     string `config:"name:env_prefix;mode:cfg;default:bf;desc:Lorem ipsum"`
		ErrMode    string `config:"name:err_mode;mode:ZZZ"`
		Skipped    string
		Nested     struct {
			Int       int `config:"name:int"`
			NestedTwo struct {
				Bool   bool   `config:"name:nestedtwo.bool"`
				String string `config:"name:string"`
				Ignore string
			} `config:"mode:cli"`
		} `config:"name:nested;mode:cli,env"`
		NestedErr struct {
			Err string `config:"name:nested.err;mode:cfg"`
		} `config:"mode:cli"`
	}
	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		field  reflect.StructField
		parent *structField
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]*structField
		wantErr bool
	}{
		{
			name:    "file",
			fields:  fields{in: &str{}, fields: make(map[string]*structField)},
			args:    args{field: reflect.ValueOf(&str{}).Elem().Type().Field(0)},
			want:    map[string]*structField{"ConfigFile": {name: "ConfigFile", tags: structFieldTags{name: "config_file", mode: modeCli, description: "Lorem ipsum", hasDescription: true}}},
			wantErr: false,
		},
		{
			name:    "env",
			fields:  fields{in: &str{}, fields: make(map[string]*structField)},
			args:    args{field: reflect.ValueOf(&str{}).Elem().Type().Field(1)},
			want:    map[string]*structField{"Prefix": {name: "Prefix", tags: structFieldTags{name: "env_prefix", mode: modeCfg, defaultValue: "bf", hasDefaultValue: true, description: "Lorem ipsum", hasDescription: true}}},
			wantErr: false,
		},
		{
			name:    "mode",
			fields:  fields{in: &str{}, fields: make(map[string]*structField)},
			args:    args{field: reflect.ValueOf(&str{}).Elem().Type().Field(2)},
			want:    map[string]*structField{},
			wantErr: true,
		},
		{
			name:    "skipped",
			fields:  fields{in: &str{}, fields: make(map[string]*structField)},
			args:    args{field: reflect.ValueOf(&str{}).Elem().Type().Field(3)},
			want:    map[string]*structField{},
			wantErr: false,
		},
		{
			name:   "nested",
			fields: fields{in: &str{}, fields: make(map[string]*structField)},
			args:   args{field: reflect.ValueOf(&str{}).Elem().Type().Field(4)},
			want: map[string]*structField{
				"Nested.Int":              {name: "Nested.Int", tags: structFieldTags{name: "nested.int", mode: modeCli | modeEnv}},
				"Nested.NestedTwo.Bool":   {name: "Nested.NestedTwo.Bool", tags: structFieldTags{name: "nested.nestedtwo.bool", mode: modeCli}},
				"Nested.NestedTwo.String": {name: "Nested.NestedTwo.String", tags: structFieldTags{name: "nested.string", mode: modeCli}},
			},
			wantErr: false,
		},
		{
			name:    "mode error",
			fields:  fields{in: &str{}, fields: make(map[string]*structField)},
			args:    args{field: reflect.ValueOf(&str{}).Elem().Type().Field(5)},
			want:    map[string]*structField{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			err := p.newStructField(tt.args.field, tt.args.parent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.newStructField() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := p.fields
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.newStructField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseCli(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string
	}{
		{name: "empty", args: []string{}, want: map[string]string{}},
		{name: "cmd", args: []string{"/buffbot"}, want: map[string]string{}},
		{name: "subcmd", args: []string{"/buffbot", "test"}, want: map[string]string{}},
		{name: "single bool", args: []string{"/buffbot", "test", "-t"}, want: map[string]string{"t": ""}},
		{name: "single param", args: []string{"/buffbot", "test", "-t", "t"}, want: map[string]string{"t": "t"}},
		{name: "single few param", args: []string{"/buffbot", "test", "-t", "-p"}, want: map[string]string{"t": "", "p": ""}},
		{name: "single param equal", args: []string{"/buffbot", "test", "-t=t"}, want: map[string]string{"t": "t"}},
		{name: "double bool", args: []string{"/buffbot", "test", "--param_bool"}, want: map[string]string{"param_bool": ""}},
		{name: "double param", args: []string{"/buffbot", "test", "--param_bool=/lorem"}, want: map[string]string{"param_bool": "/lorem"}},
		{name: "double param extra", args: []string{"/buffbot", "test", "--param_bool=/lorem", "ipsum"}, want: map[string]string{"param_bool": "/lorem"}},
		{name: "double few param", args: []string{"/buffbot", "test", "--param_bool=/lorem", "--p=test", "-m"}, want: map[string]string{"param_bool": "/lorem", "p": "test", "m": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{}
			p.parseCli(tt.args)
			if !reflect.DeepEqual(tt.want, p.parsedCli) {
				t.Errorf("Parser.newStructField() = %v, want %v", p.parsedCli, tt.want)
			}
		})
	}
}

func TestParser_parseCfg(t *testing.T) {
	dir := t.TempDir()

	json, err := os.CreateTemp(dir, "config_*.json")
	if err != nil {
		t.Error(err)
	}

	err = json.Chmod(0777)
	if err != nil {
		t.Error(err)
	}

	_, err = json.WriteString(`{"prefix":"100"}`)
	if err != nil {
		t.Error(err)
	}

	jsonRights, err := os.CreateTemp(dir, "config_*.json")
	if err != nil {
		t.Error(err)
	}

	err = jsonRights.Chmod(0000)
	if err != nil {
		t.Error(err)
	}

	brokenJson, err := os.CreateTemp(dir, "config_*.json")
	if err != nil {
		t.Error(err)
	}

	err = brokenJson.Chmod(0777)
	if err != nil {
		t.Error(err)
	}

	_, err = brokenJson.WriteString(`{"prefix":"100}`) // Broken JSON
	if err != nil {
		t.Error(err)
	}

	ini, err := os.CreateTemp(dir, "config_*.cfg")
	if err != nil {
		t.Error(err)
	}

	err = ini.Chmod(0777)
	if err != nil {
		t.Error(err)
	}

	_, err = ini.WriteString(`prefix = 100`)
	if err != nil {
		t.Error(err)
	}

	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "empty", args: args{path: ""}, wantErr: false},
		{name: "json", args: args{path: json.Name()}, wantErr: false},
		{name: "not exist", args: args{path: "/zzz.json"}, wantErr: true},
		{name: "json rights", args: args{path: jsonRights.Name()}, wantErr: true},
		{name: "broken file", args: args{path: "\000x"}, wantErr: true},
		{name: "broken json", args: args{path: brokenJson.Name()}, wantErr: true},
		{name: "ini", args: args{path: ini.Name()}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			if err := p.parseCfg(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("Parser.parseCfg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_saveToParsed(t *testing.T) {
	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		tmp    map[string]interface{}
		prefix string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]string
	}{
		{
			name: "simple",
			fields: fields{
				parsedCfg: make(map[string]string),
			},
			args: args{
				tmp: map[string]interface{}{
					"bool":   true,
					"int":    123,
					"string": "test",
					"nested": map[string]interface{}{
						"bool":   false,
						"int":    321,
						"string": "west",
						"nested": map[string]interface{}{
							"one":  1,
							"more": "123",
						},
					},
				},
			},
			want: map[string]string{
				"bool":               "true",
				"int":                "123",
				"string":             "test",
				"nested.bool":        "false",
				"nested.int":         "321",
				"nested.string":      "west",
				"nested.nested.one":  "1",
				"nested.nested.more": "123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			p.saveToParsed(tt.args.tmp, tt.args.prefix)
			if !reflect.DeepEqual(tt.want, p.parsedCfg) {
				t.Errorf("Parser.getConfig() got = %v, want %v", p.parsedCfg, tt.want)
			}
		})
	}
}

func TestParser_getConfig(t *testing.T) {
	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		name string
		mode int
	}

	cli := map[string]string{"key": "value1"}
	cfg := map[string]string{"key": "value2"}

	t.Setenv("ONE_KEY", "value3")
	t.Setenv("TWO_KEY", "value4")

	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
		want1  bool
	}{
		{name: "404", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "way", mode: 0}, want: "", want1: false},
		{name: "all", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: 0}, want: "value1", want1: true},
		{name: "cli", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeCli}, want: "value1", want1: true},
		{name: "cfg", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeCfg}, want: "value2", want1: true},
		{name: "env", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeEnv}, want: "value3", want1: true},
		{name: "cli cfg", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeCli | modeCfg}, want: "value1", want1: true},
		{name: "cli env", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeCli | modeEnv}, want: "value1", want1: true},
		{name: "cfg env", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: modeCfg | modeEnv}, want: "value2", want1: true},
		{name: "no cli", fields: fields{parsedCli: map[string]string{}, parsedCfg: cfg, envPrefix: "one_"}, args: args{name: "key", mode: 0}, want: "value2", want1: true},
		{name: "no cfg", fields: fields{parsedCli: cli, parsedCfg: map[string]string{}, envPrefix: "one_"}, args: args{name: "key", mode: 0}, want: "value1", want1: true},
		{name: "no env", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "one"}, args: args{name: "key", mode: 0}, want: "value1", want1: true},
		{name: "prefix env", fields: fields{parsedCli: cli, parsedCfg: cfg, envPrefix: "two_"}, args: args{name: "key", mode: modeEnv}, want: "value4", want1: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			got, got1 := p.getConfig(tt.args.name, tt.args.mode)
			if got != tt.want {
				t.Errorf("Parser.getConfig() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Parser.getConfig() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_writeValueToField(t *testing.T) {
	type fields struct {
		in        interface{}
		fields    map[string]*structField
		envPrefix string
		parsedCfg map[string]string
		parsedCli map[string]string
	}
	type args struct {
		key   string
		value string

		VarBool          bool
		VarInt           int
		VarInt8          int8
		VarInt16         int16
		VarInt32         int32
		VarInt64         int64
		VarUint          uint
		VarUint8         uint8
		VarUint16        uint16
		VarUint32        uint32
		VarUint64        uint64
		VarUintptr       uintptr
		VarFloat32       float32
		VarFloat64       float64
		VarComplex64     complex64
		VarComplex128    complex128
		VarArray         [5]bool
		VarChan          chan<- bool
		VarFunc          func()
		VarInterface     interface{}
		VarMap           map[int]string
		VarPointer       *bool
		VarSlice         []byte
		VarString        string
		VarStruct        struct{}
		VarUnsafePointer unsafe.Pointer
	}

	type Test struct {
		name    string
		fields  fields
		args    args
		want    func(Test) bool
		wantErr bool
	}

	tests := []Test{
		{name: "bool t", fields: fields{}, args: args{key: "VarBool", value: "t"}, want: func(t Test) bool { return t.args.VarBool == true }, wantErr: false},
		{name: "bool f", fields: fields{}, args: args{key: "VarBool", value: "f"}, want: func(t Test) bool { return t.args.VarBool == false }, wantErr: false},
		{name: "int", fields: fields{}, args: args{key: "VarInt", value: strconv.Itoa(math.MaxInt)}, want: func(t Test) bool { return t.args.VarInt == math.MaxInt }, wantErr: false},
		{name: "int err", fields: fields{}, args: args{key: "VarInt", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "int8", fields: fields{}, args: args{key: "VarInt8", value: strconv.Itoa(math.MaxInt8)}, want: func(t Test) bool { return t.args.VarInt8 == math.MaxInt8 }, wantErr: false},
		{name: "int8 err", fields: fields{}, args: args{key: "VarInt8", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "int16", fields: fields{}, args: args{key: "VarInt16", value: strconv.Itoa(math.MaxInt16)}, want: func(t Test) bool { return t.args.VarInt16 == math.MaxInt16 }, wantErr: false},
		{name: "int16 err", fields: fields{}, args: args{key: "VarInt16", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "int32", fields: fields{}, args: args{key: "VarInt32", value: strconv.Itoa(math.MaxInt32)}, want: func(t Test) bool { return t.args.VarInt32 == math.MaxInt32 }, wantErr: false},
		{name: "int32 err", fields: fields{}, args: args{key: "VarInt32", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "int64", fields: fields{}, args: args{key: "VarInt64", value: strconv.Itoa(math.MaxInt64)}, want: func(t Test) bool { return t.args.VarInt64 == math.MaxInt64 }, wantErr: false},
		{name: "int64 err", fields: fields{}, args: args{key: "VarInt64", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uint", fields: fields{}, args: args{key: "VarUint", value: "18446744073709551615"}, want: func(t Test) bool { return t.args.VarUint == math.MaxUint }, wantErr: false},
		{name: "uint err", fields: fields{}, args: args{key: "VarUint", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uint8", fields: fields{}, args: args{key: "VarUint8", value: strconv.Itoa(math.MaxUint8)}, want: func(t Test) bool { return t.args.VarUint8 == math.MaxUint8 }, wantErr: false},
		{name: "uint8 err", fields: fields{}, args: args{key: "VarUint8", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uint16", fields: fields{}, args: args{key: "VarUint16", value: strconv.Itoa(math.MaxUint16)}, want: func(t Test) bool { return t.args.VarUint16 == math.MaxUint16 }, wantErr: false},
		{name: "uint16 err", fields: fields{}, args: args{key: "VarUint16", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uint32", fields: fields{}, args: args{key: "VarUint32", value: strconv.Itoa(math.MaxUint32)}, want: func(t Test) bool { return t.args.VarUint32 == math.MaxUint32 }, wantErr: false},
		{name: "uint32 err", fields: fields{}, args: args{key: "VarUint32", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uint64", fields: fields{}, args: args{key: "VarUint64", value: "18446744073709551615"}, want: func(t Test) bool { return t.args.VarUint64 == math.MaxUint64 }, wantErr: false},
		{name: "uint64 err", fields: fields{}, args: args{key: "VarUint64", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "uintptr", fields: fields{}, args: args{key: "VarUintptr", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "float32", fields: fields{}, args: args{key: "VarFloat32", value: strconv.FormatFloat(math.MaxFloat32, 'e', -1, 32)}, want: func(t Test) bool { return t.args.VarFloat32 == math.MaxFloat32 }, wantErr: false},
		{name: "float32 err", fields: fields{}, args: args{key: "VarFloat32", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "float64", fields: fields{}, args: args{key: "VarFloat64", value: strconv.FormatFloat(math.MaxFloat64, 'e', -1, 64)}, want: func(t Test) bool { return t.args.VarFloat64 == math.MaxFloat64 }, wantErr: false},
		{name: "float64 err", fields: fields{}, args: args{key: "VarFloat64", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "complex64", fields: fields{}, args: args{key: "VarComplex64", value: strconv.FormatComplex(complex(math.MaxFloat32, math.MaxFloat32), 'f', -1, 64)}, want: func(t Test) bool { return t.args.VarComplex64 == complex(math.MaxFloat32, math.MaxFloat32) }, wantErr: false},
		{name: "complex64 err", fields: fields{}, args: args{key: "VarComplex64", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "complex128", fields: fields{}, args: args{key: "VarComplex128", value: strconv.FormatComplex(complex(math.MaxFloat64, math.MaxFloat64), 'f', -1, 128)}, want: func(t Test) bool { return t.args.VarComplex128 == complex(math.MaxFloat64, math.MaxFloat64) }, wantErr: false},
		{name: "complex128 err", fields: fields{}, args: args{key: "VarComplex128", value: "ZZZ"}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "array", fields: fields{}, args: args{key: "VarArray", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "chan", fields: fields{}, args: args{key: "VarChan", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "func", fields: fields{}, args: args{key: "VarFunc", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "interface", fields: fields{}, args: args{key: "VarInterface", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "map", fields: fields{}, args: args{key: "VarMap", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "pointer", fields: fields{}, args: args{key: "VarPointer", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "slice", fields: fields{}, args: args{key: "VarSlice", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "string", fields: fields{}, args: args{key: "VarString", value: "FDSfsdfasdfsDfe62 sd fsf4t"}, want: func(t Test) bool { return t.args.VarString == "FDSfsdfasdfsDfe62 sd fsf4t" }, wantErr: false},
		{name: "struct", fields: fields{}, args: args{key: "VarStruct", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
		{name: "unsafepointer", fields: fields{}, args: args{key: "VarUnsafePointer", value: ""}, want: func(t Test) bool { return true }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				in:        tt.fields.in,
				fields:    tt.fields.fields,
				envPrefix: tt.fields.envPrefix,
				parsedCfg: tt.fields.parsedCfg,
				parsedCli: tt.fields.parsedCli,
			}
			if err := p.writeValueToField(reflect.ValueOf(&tt.args).Elem().FieldByName(tt.args.key), tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Parser.writeValueToField() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.want(tt) {
				t.Errorf("Parser.getConfig() want wrong. Got: %v", tt.args)
			}
		})
	}
}
