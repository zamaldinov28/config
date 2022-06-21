package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

// Struct where stored all received and parsed values
type Parser struct {
	in        interface{}
	fields    map[string]*structField
	envPrefix string
	parsedCfg map[string]string // File
	parsedCli map[string]string // Command-line args
}

// Each field of received config struct has own instance
type structField struct {
	name string
	tags structFieldTags
}

// Parsed values of specific field's tags
type structFieldTags struct {
	name            string
	mode            int
	defaultValue    string
	hasDefaultValue bool
	description     string
}

const (
	// Separator that used in tags to divide different params
	separator = ";"
	// Splitter between tag param's key and value. Ex.: `name:param_name`
	separatorInner = ":"
	// Splitter between values list. Ex.: `mode:cli,cfg`
	separatorList = ","
)

// Moved to const just to have all of them at one place
const (
	tag        = "config"
	tagName    = "name"
	tagMode    = "mode"
	tagDefault = "default"
	tagDesc    = "desc"
)

// Available modes where specific param will be looked for
const (
	modeCli = 0b100
	modeCfg = 0b010
	modeEnv = 0b001
	modeAll = 0b111
)

// Keys - available modes textual values and flags
var modes = map[string]int{
	"cli": modeCli,
	"cfg": modeCfg,
	"env": modeEnv,
}

// Accepted values for boolean fields.
// While compare given value will be lowercased
var boolValues = map[bool][]string{
	true:  {"true", "t", "y", "yes"},
	false: {"false", "f", "n", "no"},
}

// Create new instance of parser for specific config struct.
func NewParser(in interface{}) (Parser, error) {
	if reflect.Pointer != reflect.ValueOf(in).Type().Kind() {
		return Parser{}, errors.New("in should be a pointer to struct")
	}

	var p = Parser{
		in:     in,
		fields: make(map[string]*structField),
	}

	// Parse struct into fields with tags
	s := reflect.ValueOf(p.in).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		field, err := p.newStructField(typeOfT.Field(i))
		if err != nil {
			return Parser{}, err
		}

		if field == nil {
			continue
		}

		p.fields[field.name] = field
	}

	return p, nil
}

// Return string with formatted and sorted usage hint
func (p *Parser) Help(prefix string) string {
	longestParameter := 0
	fieldsHelp := [][2]string{}

	for _, field := range p.fields {
		defaultHint := ""
		if field.tags.hasDefaultValue {
			defaultHint = fmt.Sprintf("[=%s]", field.tags.defaultValue)
		}
		var leftPart = fmt.Sprintf("--%s%s", field.tags.name, defaultHint)
		var rightPart = field.tags.description
		if field.tags.mode > 0 && field.tags.mode < modeAll {
			fieldModes := []string{}
			for title, mode := range modes {
				if field.tags.mode&mode > 0 {
					fieldModes = append(fieldModes, title)
				}
			}
			if len(fieldModes) > 0 {
				rightPart = fmt.Sprintf("%s (%s only)", rightPart, strings.Join(fieldModes, ", "))
			}
		}
		fieldsHelp = append(fieldsHelp, [2]string{
			leftPart,
			rightPart,
		})

		if len(leftPart) > longestParameter {
			longestParameter = len(leftPart)
		}
	}

	sort.Slice(fieldsHelp, func(i, j int) bool {
		return sort.StringsAreSorted([]string{fieldsHelp[i][0], fieldsHelp[j][0]})
	})

	buffer := bytes.NewBufferString("")
	for _, field := range fieldsHelp {
		buffer.WriteString(fmt.Sprintf("%s%-*s %s\n", prefix, longestParameter, field[0], field[1]))
	}

	return buffer.String()
}

// Execute parsing from all available sources
// Set cfgPathConfig if you use config file
// Set envPrefixConfig if you use environment variables and they have project-specific prefix.
func (p *Parser) Parse(cfgPathConfig, envPrefixConfig string) error {
	p.parseCli(os.Args)

	// Special configs that should be loaded just from cli and firstly
	for _, field := range p.fields {
		if cfgPathConfig == field.tags.name {
			if val, ok := p.getConfig(field.tags.name, field.tags.mode); ok {
				err := p.parseCfg(val)
				if err != nil {
					return err
				}
			} else if field.tags.hasDefaultValue {
				err := p.parseCfg(field.tags.defaultValue)
				if err != nil {
					return err
				}
			}
		}
		if envPrefixConfig == field.tags.name {
			if val, ok := p.getConfig(field.tags.name, field.tags.mode); ok {
				p.envPrefix = val
			} else if field.tags.hasDefaultValue {
				p.envPrefix = field.tags.defaultValue
			}
		}
	}

	s := reflect.ValueOf(p.in).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)

		parsedField, _ := p.fields[typeOfT.Field(i).Name]
		if parsedField == nil {
			continue
		}

		value, isSet := p.getConfig(parsedField.tags.name, parsedField.tags.mode)
		if !isSet {
			if parsedField.tags.hasDefaultValue {
				value = parsedField.tags.defaultValue
			} else {
				continue
			}
		}

		err := p.writeValueToField(field, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// Generate instance of structField from reflect struct field
func (p *Parser) newStructField(field reflect.StructField) (*structField, error) {
	var result = &structField{}
	result.name = field.Name

	tagValue, ok := field.Tag.Lookup(tag)
	if !ok {
		return nil, nil
	}

	tags := strings.Split(tagValue, separator)
	for _, flag := range tags {
		tmp := strings.Split(flag, separatorInner)
		fieldTagName := tmp[0]
		fieldTagValue := strings.Join(tmp[1:], separatorInner)
		switch fieldTagName {
		case tagName:
			result.tags.name = fieldTagValue
		case tagMode:
			result.tags.mode = 0
			listTmp := strings.Split(fieldTagValue, separatorList)
			for _, val := range listTmp {
				key, ok := modes[val]
				if !ok {
					return nil, errors.New(fmt.Sprintf("Unknown mode %s. Available modes: %s", val, strings.Join(maps.Keys(modes), ", ")))
				}
				result.tags.mode = result.tags.mode | key
			}
		case tagDefault:
			result.tags.defaultValue = fieldTagValue
			result.tags.hasDefaultValue = true
		case tagDesc:
			result.tags.description = fieldTagValue
		}
	}

	return result, nil
}

// Parse arguments from command line
func (p *Parser) parseCli(args []string) {
	p.parsedCli = make(map[string]string)
	pendingName := ""
	for _, arg := range args {
		if '-' != arg[0] {
			if "" != pendingName {
				p.parsedCli[pendingName] = arg
				pendingName = ""
			}
			continue
		}

		if '-' == arg[0] && "" != pendingName {
			p.parsedCli[pendingName] = ""
			pendingName = ""
		}

		tmp := strings.Split(arg, "=")
		name := strings.TrimLeft(tmp[0], "-")

		if len(tmp) == 1 {
			pendingName = name
			continue
		}

		p.parsedCli[name] = strings.Join(tmp[1:], "=")
	}

	if "" != pendingName {
		p.parsedCli[pendingName] = ""
	}
}

// Read and parse config file
func (p *Parser) parseCfg(path string) error {
	p.parsedCfg = make(map[string]string)

	if "" == path {
		return nil
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return errors.New("Cannot find config file")
	} else if err != nil {
		return err
	}

	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	ext := filepath.Ext(path)

	if ".json" == ext {
		tmp := make(map[string]interface{})
		err = json.Unmarshal(fileContent, &tmp)
		if err != nil {
			return err
		}

		for k, v := range tmp {
			p.parsedCfg[k] = fmt.Sprint(v)
		}

		return nil
	}

	return nil
}

// Look for specific config in allowed (for this field) places
func (p *Parser) getConfig(name string, mode int) (string, bool) {
	var value = ""
	var find = false

	if 0 == mode || mode&modeEnv > 0 {
		if tmpValue, ok := os.LookupEnv(strings.ToUpper(fmt.Sprintf("%s%s", p.envPrefix, name))); ok {
			value = tmpValue
			find = true
		}
	}

	if 0 == mode || mode&modeCfg > 0 {
		if tmpValue, ok := p.parsedCfg[name]; ok {
			value = tmpValue
			find = true
		}
	}

	if 0 == mode || mode&modeCli > 0 {
		if tmpValue, ok := p.parsedCli[name]; ok {
			value = tmpValue
			find = true
		}
	}

	return value, find
}

// Convert founded value to required type, and put it into struct field
func (p *Parser) writeValueToField(field reflect.Value, value string) error {
	switch field.Type().Kind() {
	case reflect.Bool:
		value = strings.ToLower(value)
	Exit:
		for b, words := range boolValues {
			for _, word := range words {
				if value == word {
					field.SetBool(b)
					break Exit
				}
			}
		}
	case reflect.Int:
		convValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(convValue)
	case reflect.Int8:
		convValue, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			return err
		}
		field.SetInt(convValue)
	case reflect.Int16:
		convValue, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return err
		}
		field.SetInt(convValue)
	case reflect.Int32:
		convValue, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return err
		}
		field.SetInt(convValue)
	case reflect.Int64:
		convValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(convValue)
	case reflect.Uint:
		convValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(convValue)
	case reflect.Uint8:
		convValue, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return err
		}
		field.SetUint(convValue)
	case reflect.Uint16:
		convValue, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return err
		}
		field.SetUint(convValue)
	case reflect.Uint32:
		convValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		field.SetUint(convValue)
	case reflect.Uint64:
		convValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(convValue)
	case reflect.Float32:
		return errors.New("Float32 are not supported yet")
	case reflect.Float64:
		return errors.New("Float64 are not supported yet")
	case reflect.Complex64:
		return errors.New("Complex64 are not supported yet")
	case reflect.Complex128:
		return errors.New("Complex128 are not supported yet")
	case reflect.Array:
		return errors.New("Array are not supported yet")
	case reflect.Chan:
		return errors.New("Chan are not supported yet")
	case reflect.Map:
		return errors.New("Map are not supported yet")
	case reflect.Slice:
		return errors.New("Slice are not supported yet")
	case reflect.String:
		field.SetString(value)
	case reflect.Struct:
		return errors.New("Struct are not supported yet")
	default: // Uintptr, Func, Interface, Pointer, UnsafePointer
		return errors.New(fmt.Sprintf("%s is not supported", field.Type().String()))
	}

	return nil
}
