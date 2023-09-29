package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"time"

	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	"gopkg.in/yaml.v3"
)

var (
	DEFAULT_LOGLEVEL = log.InfoLevel
	STATE_MAP        = map[string]string{
		"[]":  "OPEN",
		"[ ]": "OPEN",
		"[-]": "IN_PROGRESS",
		"[/]": "IN_PROGRESS",
		"[x]": "DONE",
		"[X]": "DONE",
	}
)

type Granite struct {
	RootPath           string
	TemplateConfigPath string
	TodoTag            string
	logger             *log.Logger
}

type Todo struct {
	// RawLine is the complete unparse line from the markdown file
	RawLine string
	// Text contains the todo string without leading whitespace and -
	Text string `json:"text"`
	// Tags are the tags assigned to the todo
	Tags []string `json:"tags"`
	// LineNumber is the line in the file where the todo was found
	LineNumber int `json:"lnum"`
	// FilePath is the path to the file where the todo was found
	FilePath string `json:"filename"`
	// DueDate is the date when the todo is due. The format must be YYYY-MM-DD
	DueDate string `json:"due_date"`
	// StateString is a string representation of the State of the ToDo, eg: OPEN, DONE, IN_PROGRESS
	StateString string `json:"state"`
}

// Parse parses the RawLine set in the todo and populates all other fields based on what it finds there.
//
// Returns an error when RawLine, LineNumber or FilePath are unset or the RawLine can't be parsed into a todo
func (t *Todo) Parse() error {
	stateRegex := regexp.MustCompile(`(\[.?\])`)
	tagsRegex := regexp.MustCompile(`(#\w+)`)
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	textRegex := regexp.MustCompile(`^.*(\[.?\].*)$`)

	stateRaw := stateRegex.FindString(t.RawLine)
	stateString, ok := STATE_MAP[stateRaw]
	if !ok {
		return fmt.Errorf("Couldn't parse todo state: '%s', line is: '%s'", stateRaw, t.RawLine)
	}
	t.StateString = stateString
	t.Tags = tagsRegex.FindAllString(t.RawLine, 100)
	if len(t.Tags) == 0 {
		return fmt.Errorf("Couldn't parse todo Tags: %s", t.RawLine)
	}

	t.DueDate = dateRegex.FindString(t.RawLine)
	t.Text = textRegex.FindStringSubmatch(t.RawLine)[1]
	return nil
}

func (g *Granite) GetTodos(args []string) (string, error) {
	g.logger.Debugf("Called GetTodos with args: %v", args)
	mdFiles, err := GetAllFilesWithExtInDir(g.RootPath, ".md")

	if err != nil {
		g.logger.Errorf("Error getting Todos: %v", err)
		return "", err
	}

	todos := []*Todo{}

	for _, mdFilePath := range mdFiles {
		mdFile, err := os.Open(mdFilePath)
		if err != nil {
			g.logger.Warnf("Error opening file %s for reading: %v. Skipping.", mdFilePath, err)
			continue
		}
		scanner := bufio.NewScanner(mdFile)
		scanner.Split(bufio.ScanLines)
		lineNumber := 1
		if !scanner.Scan() {
			g.logger.Infof("Scanning finished prematurely: %s", mdFilePath)
			continue
		}
		for line := scanner.Text(); scanner.Scan(); line = scanner.Text() {
			if strings.Contains(line, g.TodoTag) {
				t := &Todo{
					RawLine:    line,
					LineNumber: lineNumber,
					FilePath:   mdFilePath,
				}
				err := t.Parse()
				if err == nil {
					todos = append(todos, t)
				} else {
					g.logger.Infof("Error parsing Log Line '%s': %v", line, err)
				}
			}
			lineNumber++
		}
	}

	rawJson, err := json.Marshal(todos)

	return string(rawJson), err
}

func GetAllFilesWithExtInDir(dir string, ext string) ([]string, error) {
	foundFiles := []string{}
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(path, ext) {
				return nil
			}
			foundFiles = append(foundFiles, path)
			return nil
		})

	return foundFiles, err
}

type TemplateConfig struct {
	Name         string   `yaml:"name" json:"name"`
	Path         string   `yaml:"path" json:"path"`
	Parameters   []string `yaml:"parameters" json:"parameters"`
	OutputFolder string   `yaml:"output_folder" json:"output_folder"`
}

type RenderTemplateInput struct {
  Template TemplateConfig `json:"template"`
  Options map[string]string `json:"options"` 
}

func (g *Granite) RenderTemplate(args []string) (string, error) {
	g.logger.Debugf("Called RenderTemplate with args: %v", args)
  input := &RenderTemplateInput{}

	if len(args) != 1 {
		g.logger.Errorf("RenderTemplate expects 1 argument, got: %d", len(args))
		return "", fmt.Errorf("RenderTemplate expects 1 argument, got: %d", len(args))
	}
  
  err := json.Unmarshal([]byte(args[0]), input)

  return "", nil
}

func (g *Granite) GetTemplates(args []string) (string, error) {
	g.logger.Debugf("Called GetTemplates with args: %v", args)
	templateConfigRaw, err := ioutil.ReadFile(path.Join(g.RootPath, g.TemplateConfigPath))
	g.logger.Debugf("got template config raw %s",templateConfigRaw )
	if err != nil {
		log.Errorf("Error opening template config file for reading: %v", err)
		return "", fmt.Errorf("Error opening template config file for reading: %w", err)
	}

	templateConfigs := []TemplateConfig{}

	err = yaml.Unmarshal(templateConfigRaw, &templateConfigs)
	if err != nil {
		log.Errorf("Error parsing template config file %s: %v", g.TemplateConfigPath, err)
		return "", fmt.Errorf(
			"Error parsing template config file %s: %w",
			g.TemplateConfigPath,
			err,
		)
	}
	log.Infoln("Successfully parsed template config")

	return "", nil
}

type InitArgs struct {
	RootPath           string
	TemplateConfigPath string
	TodoTag            string
	LogLevel           string `json:",omitempty"`
}

func (g *Granite) Init(args []string) (string, error) {
	g.logger.Debugf("Called Init with args: %v", args)

	if len(args) != 1 {
		g.logger.Errorf("Init expects 1 argument, got: %d", len(args))
		return "", fmt.Errorf("Init expects 1 argument, got: %d", len(args))
	}

	initargs := &InitArgs{}

	err := json.Unmarshal([]byte(args[0]), initargs)
	if err != nil {
		g.logger.Errorf("Couln't parse json data: %v", err)
		return "", fmt.Errorf("Couln't parse json data: %w", err)
	}

	g.TodoTag = initargs.TodoTag
	g.TemplateConfigPath = initargs.TemplateConfigPath
	g.RootPath = initargs.RootPath

	var logLevel log.Level = DEFAULT_LOGLEVEL
	if initargs.LogLevel != "" {
		logLevel, err = log.ParseLevel(initargs.LogLevel)
		if err != nil {
			log.Warnf("Couldn't parse loglevel %s: %v", initargs.LogLevel, err)
			return "", fmt.Errorf("Couldn't parse loglevel %s: %w", initargs.LogLevel, err)
		}
	}

	g.logger.SetLevel(logLevel)
	return "Called Init", nil
}

func main() {
	f, err := os.OpenFile(
		"/home/maschulz/granite_go.log",
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("Can't open log file")
	}
	graniteLogger := log.New()
	graniteLogger.SetOutput(f)
	graniteLogger.SetFormatter(&log.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: false,
		FullTimestamp:    false,
		TimestampFormat:  time.RFC3339,
		QuoteEmptyFields: true,
	})

	defer f.Close()
	g := &Granite{
		logger: graniteLogger,
	}

	g.logger.Println("Logger init done")
	plugin.Main(func(p *plugin.Plugin) error {

		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetTodos"}, g.GetTodos)
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetTemplates"}, g.GetTemplates)
		p.HandleFunction(&plugin.FunctionOptions{
			Name: "GraniteInit",
		}, g.Init)
		return nil
	})
	g.logger.Infof("after plugin init")
}

// 1. init with settings
