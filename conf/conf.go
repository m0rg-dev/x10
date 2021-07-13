package conf

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"m0rg.dev/x10/x10_log"
)

var config = map[string]string{}

func ReadConfig(path string) {
	logger := x10_log.Get("readconfig")
	paths := strings.Split(path, ":")
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			logger.Warn(err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			str := scanner.Text()
			split := strings.Split(str, "=")
			split[0] = strings.TrimSpace(split[0])
			split[1] = strings.TrimSpace(strings.Join(split[1:], "="))
			config[split[0]] = split[1]
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	}
}

func Get(key string) string {
	str, ok := config[key]
	if !ok {
		panic(fmt.Errorf("unknown configuration key: %s", key))
	}
	return str
}

func GetBool(key string) bool {
	str, ok := config[key]
	if !ok {
		panic(fmt.Errorf("unknown configuration key: %s", key))
	}
	return str == "true"
}

// func Set(key string, val string) {
// 	config[key] = val
// }

type ConfigKey struct {
	HelpText   string
	TakesValue bool
	Default    string
}

var known_keys = map[string]ConfigKey{}
var command_list = map[string]string{}

func RegisterKey(command string, key string, meta ConfigKey) {
	if command != "" {
		key = command + ":" + key
	}

	known_keys[key] = meta
	config[key] = meta.Default
}

func RegisterCommand(command string, synopsis string) {
	command_list[command] = synopsis
}

func ParseError(reason string) {
	logger := x10_log.Get("command-line")
	logger.Fatalf("Error while parsing command-line arguments: %s", reason)
}

func AssertArgumentCount(command string, count int, args []string) {
	if len(args) != count {
		ParseError(command + " subcommand expects " + strconv.Itoa(count) + " arguments.")
	}
}

func ParseCommandLine(args []string) (command string, additional_args []string) {
	options := []string{}
	// pass 1: find command and relevant options
	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		// stop parsing on --
		if arg == "--" {
			additional_args = append(additional_args, args...)
			break
		} else if arg == "--help" {
			if command == "" {
				PrintHelpText(nil)
			} else {
				PrintHelpText(&command)
			}
		} else if strings.HasPrefix(arg, "--") {
			opt := strings.TrimPrefix(arg, "--")
			if command != "" {
				opt = command + ":" + opt
			}

			if strings.Contains(opt, "=") {
				options = append(options, opt)
			} else {
				meta, ok := known_keys[opt]
				if !ok {
					ParseError("Unknown command-line option: " + opt)
				}
				if meta.TakesValue {
					if len(args) == 0 {
						ParseError("Option " + opt + " requires an argument")
					}
					opt = opt + "=" + args[0]
					args = args[1:]
					options = append(options, opt)
				}
			}
		} else if command == "" {
			command = arg
		} else {
			additional_args = append(additional_args, arg)
		}
	}

	// pass 2: process into config map
	for _, opt := range options {
		if strings.Contains(opt, "=") {
			s := strings.SplitN(opt, "=", 2)
			key := s[0]
			value := s[1]
			meta, ok := known_keys[key]
			if !ok {
				ParseError("Unknown command-line option: " + key)
			}
			if !meta.TakesValue {
				ParseError("Option " + opt + " does not take an argument")
			}

			config[key] = value
		} else {
			value := "true"
			if strings.HasPrefix(opt, "no-") {
				value = "false"
				opt = strings.TrimPrefix(opt, "no-")
			}
			meta, ok := known_keys[opt]
			if !ok {
				ParseError("Unknown command-line option: " + opt)
			}
			if meta.TakesValue {
				ParseError("Option " + opt + " requires an argument")
			}

			config[opt] = value
		}
	}

	if command == "" {
		PrintHelpText(nil)
	}

	return command, additional_args
}

func PrintHelpText(command *string) {
	keys_generic := []string{}
	keys_command := []string{}

	for key := range known_keys {
		if strings.Contains(key, ":") {
			s := strings.SplitN(key, ":", 2)
			cmd := s[0]
			key = s[1]
			if command != nil && cmd == *command {
				keys_command = append(keys_command, key)
			}
		} else {
			keys_generic = append(keys_generic, key)
		}
	}

	if command == nil {
		fmt.Fprintln(os.Stderr, "Usage: "+os.Args[0]+" [options] <subcommand> [subcommand options] [subcommand args...]")
	} else {
		fmt.Fprintln(os.Stderr, "Usage: "+os.Args[0]+" [options] "+*command+" "+command_list[*command])
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options common to all commands:")

	sort.Strings(keys_generic)
	for _, key := range keys_generic {
		meta := known_keys[key]
		if meta.TakesValue {
			fmt.Fprintf(os.Stderr,
				"  %40s   %s\n", fmt.Sprintf("--%s=%s", key, meta.Default),
				meta.HelpText)
		} else {
			if meta.Default == "true" {
				key = "no-" + key
			}
			fmt.Fprintf(os.Stderr,
				"  %40s   %s\n", fmt.Sprintf("--%s", key),
				meta.HelpText)
		}
	}

	fmt.Fprintln(os.Stderr, "")

	if command == nil {
		fmt.Fprintln(os.Stderr, "Available commands: ")
		l := []string{}
		for cmd := range command_list {
			l = append(l, cmd)
		}
		sort.Strings(l)

		for _, cmd := range l {
			fmt.Fprintf(os.Stderr, "  %s\n", cmd)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Options specific to the "+*command+" subcommand:")

		sort.Strings(keys_command)
		for _, key := range keys_command {
			meta := known_keys[*command+":"+key]
			if meta.TakesValue {
				fmt.Fprintf(os.Stderr,
					"  %40s   %s\n", fmt.Sprintf("--%s=%s", key, meta.Default),
					meta.HelpText)
			} else {
				if meta.Default == "true" {
					key = "no-" + key
				}
				fmt.Fprintf(os.Stderr,
					"  %40s   %s\n", fmt.Sprintf("--%s", key),
					meta.HelpText)
			}
		}
	}

	os.Exit(1)
}

// TODO target profiles
func TargetDir() string {
	panic("NYI")
}
