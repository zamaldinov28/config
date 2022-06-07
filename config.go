package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

type parser struct {
	in        interface{}
	fields    map[string]structField
	envPrefix string
	cfgPath   string
	parsedCfg map[string]string // File
	parsedCli map[string]string // Command-line args
}

type structField struct {
	name      string
	fieldType string
	value     string
	tags      structFieldTags
}

type structFieldTags struct {
	name            string
	mode            int
	defaultValue    string
	hasDefaultValue bool
	description     string
	flags           int
}

const separator = ";"
const separatorInner = ":"
const separatorList = ","

const (
	tag        = "config"
	tagName    = "name"
	tagMode    = "mode"
	tagDefault = "default"
	tagDesc    = "desc"
	tagFlag    = "flag"
)

const (
	modeCli = 0b100
	modeCfg = 0b010
	modeEnv = 0b001
)

var modes = map[string]int{
	"cli": modeCli,
	"cfg": modeCfg,
	"env": modeEnv,
}

const (
	flagConfigFile = 0b10
	flagEnvPrefix  = 0b01
)

var flags = map[string]int{
	"config_file": flagConfigFile,
	"env_prefix":  flagEnvPrefix,
}

var boolValues = map[bool][]string{
	true:  {"true", "t", "y", "yes"},
	false: {"false", "f", "n", "no"},
}

func NewParser(in interface{}) (parser, error) {
	if reflect.Pointer != reflect.ValueOf(in).Type().Kind() {
		return parser{}, errors.New("in should be a pointer to struct")
	}

	return parser{
		in: in,
	}, nil
}

func (p *parser) Parse() error {
	p.fields = make(map[string]structField)

	p.parseCli(os.Args)

	s := reflect.ValueOf(p.in).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		field, err := p.newStructField(typeOfT.Field(i))
		if err != nil {
			return err
		}

		// Special configs that should be loaded just from cli and firstly
		if field.tags.flags&flagConfigFile > 0 {
			if val, ok := p.getConfig(field.tags.name, field.tags.mode); ok {
				field.value = val
				p.cfgPath = val
			}
		}
		if field.tags.flags&flagEnvPrefix > 0 {
			if val, ok := p.getConfig(field.tags.name, field.tags.mode); ok {
				field.value = val
				p.envPrefix = val
			}
		}
		p.fields[field.name] = field
	}

	err := p.parseCfg()
	if err != nil {
		return err
	}

	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)

		parsedField, _ := p.fields[typeOfT.Field(i).Name]

		value, isSet := p.getConfig(parsedField.tags.name, parsedField.tags.mode)
		if !isSet {
			if parsedField.tags.hasDefaultValue {
				value = parsedField.tags.defaultValue
			} else {
				continue
			}
		}

		err = p.writeValueToField(field, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *parser) newStructField(field reflect.StructField) (structField, error) {
	var result = structField{}
	result.name = field.Name
	result.fieldType = field.Type.String()

	tags := strings.Split(field.Tag.Get(tag), separator)
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
					return structField{}, errors.New(fmt.Sprintf("Unknown mode %s. Available modes: %s", val, strings.Join(maps.Keys(modes), ", ")))
				}
				result.tags.mode = result.tags.mode | key
			}
		case tagDefault:
			result.tags.defaultValue = fieldTagValue
			result.tags.hasDefaultValue = true
		case tagDesc:
			result.tags.description = fieldTagValue
		case tagFlag:
			listTmp := strings.Split(fieldTagValue, separatorList)
			for _, val := range listTmp {
				key, ok := flags[val]
				if !ok {
					return structField{}, errors.New(fmt.Sprintf("Unknown flag %s. Available flags: %s", val, strings.Join(maps.Keys(flags), ", ")))
				}
				result.tags.flags = result.tags.flags | key
			}
		}
	}

	return result, nil
}

func (p *parser) parseCli(args []string) {
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

func (p *parser) parseCfg() error {
	p.parsedCfg = make(map[string]string)

	if "" == p.cfgPath {
		return nil
	}

	if _, err := os.Stat(p.cfgPath); errors.Is(err, os.ErrNotExist) {
		return errors.New("Cannot find config file")
	} else if err != nil {
		return err
	}

	fileContent, err := ioutil.ReadFile(p.cfgPath)
	if err != nil {
		return err
	}

	ext := filepath.Ext(p.cfgPath)

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

func (p *parser) getConfig(name string, mode int) (string, bool) {
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

func (p *parser) writeValueToField(field reflect.Value, value string) error {
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
