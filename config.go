package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/skeema/tengo"
	"github.com/spf13/pflag"
)

type Target struct {
	Host     string
	Port     int
	User     string
	Password string
	Schema   string
	Driver   string
	instance *tengo.Instance
}

// Returns the DSN without any trailing schema name
func (t Target) BaseDSN() string {
	var userAndPass string
	if t.Password == "" {
		userAndPass = t.User
	} else {
		userAndPass = fmt.Sprintf("%s:%s", t.User, t.Password)
	}
	return fmt.Sprintf("%s@tcp(%s:%d)/", userAndPass, t.Host, t.Port)
}

func (t Target) DSN() string {
	return t.BaseDSN() + t.Schema
}

func (t Target) HostAndOptionalPort() string {
	if t.Port == 3306 {
		return t.Host
	} else {
		return fmt.Sprintf("%s:%d", t.Host, t.Port)
	}
}

// MergeCLIConfig takes in supplied command-line flags, and merges them into the target,
// overriding any
func (t *Target) MergeCLIConfig(cliConfig *ParsedGlobalFlags) {
	if cliConfig == nil {
		return
	}
	if cliConfig.Host != "" {
		t.Host = cliConfig.Host
	}
	if cliConfig.Port != 0 {
		t.Port = cliConfig.Port
	}
	if cliConfig.User != "" {
		t.User = cliConfig.User
	}
	if cliConfig.Password != "" {
		t.Password = cliConfig.Password
	}
	if cliConfig.Schema != "" {
		t.Schema = cliConfig.Schema
	}

	if t.User == "" {
		t.User = "root"
	}
	if t.Host == "" {
		t.Host = "127.0.0.1"
	}
	if t.Port == 0 {
		parts := strings.SplitN(t.Host, ":", 2)
		if len(parts) > 1 {
			t.Host = parts[0]
			t.Port, _ = strconv.Atoi(parts[1])
		}
		if t.Port == 0 {
			t.Port = 3306
		}
	}
}

func (t *Target) Instance() *tengo.Instance {
	if t.instance == nil {
		t.instance = tengo.NewInstance(t.Driver, t.BaseDSN())
	}
	return t.instance
}

func (t *Target) DB() *sqlx.DB {
	return t.Instance().Connect(t.Schema)
}

type TargetList []*Target

// SetInstances bulk-hydrates a tengo instance for each Target. The instances
// will be de-duped, such that targets representing different schemas on the
// same physical instance will point to the same tengo.Instance.
func (tl TargetList) SetInstances() {
	dsnToInstance := make(map[string]*tengo.Instance, len(tl))
	for _, t := range tl {
		dsn := t.BaseDSN()
		if dsnToInstance[dsn] == nil {
			dsnToInstance[dsn] = tengo.NewInstance(t.Driver, dsn)
		}
		t.instance = dsnToInstance[dsn]
	}
}

type Config struct {
	GlobalFiles  []*SkeemaFile
	GlobalFlags  *ParsedGlobalFlags
	CommandFlags *pflag.FlagSet
}

type ParsedGlobalFlags struct {
	Path     string
	Host     string
	Port     int
	User     string
	Password string
	Schema   string
}

func ParseGlobalFlags(flags *pflag.FlagSet) (parsed *ParsedGlobalFlags, err error) {
	parsed = new(ParsedGlobalFlags)
	if parsed.Path, err = flags.GetString("dir"); err != nil {
		return parsed, errors.New("Invalid value for --dir option")
	}
	if parsed.Host, err = flags.GetString("host"); err != nil {
		return parsed, errors.New("Invalid value for --host option")
	}
	if parsed.Port, err = flags.GetInt("port"); err != nil {
		return parsed, errors.New("Invalid value for --port option")
	}
	if parsed.User, err = flags.GetString("user"); err != nil {
		return parsed, errors.New("Invalid value for --user option")
	}
	if parsed.Password, err = flags.GetString("password"); err != nil {
		return parsed, errors.New("Invalid value for --password option")
	}
	if parsed.Schema, err = flags.GetString("schema"); err != nil {
		return parsed, errors.New("Invalid value for --schema option")
	}
	return parsed, nil
}
