# Config setup from different sources

[![Buy me a coffee](https://badgen.net/badge/icon/buymeacoffee?icon=buymeacoffee&label)](https://www.buymeacoffee.com/zamaldinov28)
![Last stable version](https://badgen.net/github/release/zamaldinov28/config)
[![Go Reference](https://pkg.go.dev/badge/github.com/zamaldinov28/config.svg)](https://pkg.go.dev/github.com/zamaldinov28/config)

If you need to take configs from different env, config file, or from cli, combine it into one struct, and use it as simple as possible, you are here! Tiny package allow you to setup everything you want using golang tag `config`, and few lines of code.

This package was created for its own needs, but can be used by anyone. Work on it is still being carried out, so if you need some kind of feature or you want to just make this package better, feel free to write to me, or make a PR by yourself.

Pleasant use!

## Example

```golang
package main

import (
	"fmt"

	"github.com/zamaldinov28/config"
)

type Config struct {
	ConfigFile string `config:"name:config_file;mode:cli;desc:Use config from file. Should be in JSON or Unix-like configuration file format"`
	Prefix     string `config:"name:prefix;mode:cli;default:;desc:Use environment variables prefix. Ex.: (default) DB_HOST, (-prefix=CNF) CNF_DB_HOST"`

	Env     string `config:"name:env;default:dev;desc:Current environment. Available values: dev, test, stage, prod"`
	DbUser  string `config:"name:db_user;default:user;desc:Database user"`
	DbPass  string `config:"name:db_pass;default:;desc:Database password"`
	DbHost  string `config:"name:db_host;default:localhost;desc:Database host"`
	DbPort  string `config:"name:db_port;default:3306;desc:Database port"`
	DbName  string `config:"name:db_name;default:database;desc:Database name"`
	LogFile string `config:"name:log_file;desc:Path to log file. If empty logs will be sended to stdout. If path or file not exists, it will be created"`
}

func main() {
	var cfg Config
	var err error

	parser, err := config.NewParser(&cfg)
	if err != nil {
		panic(err)
	}
	err = parser.Parse()
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg)
}
```

## Directives

### `name`

Config name/key. Example:

```golang
DbUser string `config:"name:db_user"`
```

For this field with can set value with cli `--db_user=your_user`, with json file
```json
{
	"db_user": "your_user"
}
```
or by setting environment variable (depends on your OS) `DB_USER=your_user`

> Note! To take value from environment variable name will be uppercased!

### `mode`

Source of the config. Support one of the following values:

- `cli` - for command-line arguments
- `cfg` - for config file
- `env` - for environment variables

Can accept few sources, separated by ","

Example:

```golang
DbUser string `config:"name:db_user;mode:cli,cfg"`
```

### `default`

Default value for field. Example:

```golang
DbUser string `config:"name:db_user;mode:cli,cfg;default:root"`
```

For this example, if `db_name` not set with command-line neither exist in config file, the `root` value will be applied. But in case if you set empty value (ex.: `--db_name=` or `"db_name":""`) default value will be ignored.

### `desc`

Textual description of field. Uses for show help hint. Example:

```golang
DbUser string `config:"name:db_user;mode:cli,cfg;default:root;desc:Database username"`
```

will print

```
    --db_user[=root] Database username (cli, cfg only)
```

You can skip this parameter, and in this case this field will not be added to help hint. Also you can add empty description to field. In this case will be printed just auto-generated info. Example:

```golang
First  string `config:"name:first;mode:cli,cfg;default:root"`
Second string `config:"name:second;mode:cli,cfg;default:root;desc:"`
Third  string `config:"name:third;mode:env;desc:Lorem ipsum"`
```

will print

```
    --second[=root] (cli, cfg only)
    --third         Lorem ipsum (env only)
```