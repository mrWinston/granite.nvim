package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"text/template"
	"time"

	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/mrWinston/granite.nvim/pkg/codeblock"
	"github.com/mrWinston/granite.nvim/pkg/markdown"
	"github.com/mrWinston/granite.nvim/pkg/models"
	"github.com/mrWinston/granite.nvim/pkg/tagquery"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	log "github.com/sirupsen/logrus"
	ts "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
)

var (
	DEFAULT_LOGLEVEL = log.DebugLevel
)

const DEFAULT_DATE_FORMAT = "2006-01-02"
const EXTMARK_NS = "codeblock_run"

type Granite struct {
	ConfigFile string            `json:"config_file" yaml:"config_file"`
	RootPath   string            `json:"root_path" yaml:"root_path"`
	TodoTag    string            `json:"todo_tag" yaml:"todo_tag"`
	logger     *log.Logger       `json:"logger" yaml:"logger"`
	Templates  []*TemplateConfig `json:"templates" yaml:"templates"`
}

type GetTodosArgs struct {
	States   []string `json:"states,omitempty" yaml:"states,omitempty"`
	Tag      string   `json:"tag,omitempty" yaml:"tag,omitempty"`
	TagQuery string   `json:"tag_query,omitempty" yaml:"tag_query,omitempty"`
	Due      string   `json:"due,omitempty" yaml:"due,omitempty"`
}

func newTemplate(name string) *template.Template {
	additionalFunc := template.FuncMap{
		"calendarWeek": func() int {
			_, week := time.Now().ISOWeek()
			return week
		},
	}
	t := template.New(name).Funcs(sprig.FuncMap()).Funcs(additionalFunc)
	return t
}

func (g *Granite) FilterTodos(todos []*models.Todo, query string) ([]*models.Todo, error) {
	result := []*models.Todo{}
	// ( tag OR bla ) AND todo

	// split query into tokens
	// replace all tag tokens with the list of todos
	// aggregate list from inside out with AND and OR

	tokens := tagquery.QueryToTokens(query)
	g.logger.Infof("Got Tokens: [%s]", strings.Join(tokens, ","))
	tree, err := tagquery.BuildTokenTree(tokens)
	if err != nil {
		return result, err
	}

	g.logger.Infof("Got Tree: %s", tree)

	return tree.GetTodos(func(s string) []*models.Todo {
		return Filter(todos, func(element *models.Todo) bool {
			compare, negate := strings.CutPrefix(s, "!")
			contains := strings.Contains(strings.Join(element.Tags, ","), compare)

			if negate {
				return !contains
			} else {
				return contains
			}
		})
	}), nil
}

func (g *Granite) GetTodos(args []string) (string, error) {
	g.logger.Debugf("Called GetTodos with args: %v", args)
	if len(args) != 1 {
		g.logger.Errorf("GetTodos expects exactly 1 argument.")
		return "", fmt.Errorf("GetTodos expects exactly 1 argument.")
	}

	getArgs := &GetTodosArgs{}
	err := json.Unmarshal([]byte(args[0]), getArgs)
	if err != nil {
		g.logger.Warnf("Error parsing args for GetTodos: %v", err)
	}

	todos, err := g.GetAllTodos()
	if err != nil {
		g.logger.Errorf("Error getting todos from markdown files: %v", err)
		return "", fmt.Errorf("Error getting todos from markdown files: %w", err)
	}

	if len(getArgs.States) > 0 {
		getargsstatetmp := strings.Join(getArgs.States, ",")
		todos = Filter[*models.Todo](todos, func(element *models.Todo) bool {
			return strings.Contains(getargsstatetmp, element.StateString)
		})
	}

	if getArgs.TagQuery != "" {
		todos, err = g.FilterTodos(todos, getArgs.TagQuery)
		if err != nil {
			g.logger.Errorf("Error Getting todos from Query: %v", err)
			return "", fmt.Errorf("Error Getting todos from Query: %w", err)
		}
	} else if getArgs.Tag != "" {
		todos = Filter[*models.Todo](todos, func(element *models.Todo) bool {
			g.logger.Errorf("%v", element.Tags)
			return strings.Contains(strings.Join(element.Tags, ","), getArgs.Tag)
		})
	}

	if getArgs.Due != "" {
		due, err := time.Parse(DEFAULT_DATE_FORMAT, getArgs.Due)
		if err != nil {
			g.logger.Errorf("Cannot parse due date '%s' : %v", getArgs.Due, err)
			return "", fmt.Errorf("Cannot parse due date '%s' : %w", getArgs.Due, err)
		}

		todos = Filter[*models.Todo](todos, func(element *models.Todo) bool {
			todoDue, err := time.Parse(DEFAULT_DATE_FORMAT, element.DueDate)
			if err != nil {
				g.logger.Errorf("Cannot parse due date for todo '%s' : %v", element.Text, err)
				return false
			}
			return todoDue.UnixMilli() <= due.UnixMilli()
		})
	}

	rawJson, err := json.Marshal(todos)

	return string(rawJson), err
}

func Filter[T any](elements []T, filterFunc func(element T) bool) []T {
	n := []T{}
	for _, v := range elements {
		if filterFunc(v) {
			n = append(n, v)
		}
	}
	return n
}

func (g *Granite) GetAllTodosWithTag(tag string) ([]*models.Todo, error) {
	allTodos, err := g.GetAllTodos()
	if err != nil {
		return nil, err
	}
	filteredTodos := Filter[*models.Todo](allTodos, func(element *models.Todo) bool {
		return strings.Contains(strings.Join(element.Tags, ","), tag)
	})

	return filteredTodos, nil
}

func (g *Granite) GetAllTags() ([]string, error) {
	todos, err := g.GetAllTodos()
	if err != nil {
		g.logger.Errorf("Error getting todos from markdown files: %v", err)
		return nil, fmt.Errorf("Error getting todos from markdown files: %w", err)
	}
	tempTagMap := map[string]bool{}
	for _, todo := range todos {
		for _, tag := range todo.Tags {
			tempTagMap[tag] = true
		}
	}
	allTags := make([]string, len(tempTagMap))
	i := 0
	for k := range tempTagMap {
		allTags[i] = k
		i++
	}

	return allTags, nil
}

func (g *Granite) GetAllTodos() ([]*models.Todo, error) {

	mdFiles, err := GetAllFilesWithExtInDir(g.RootPath, ".md")

	if err != nil {
		g.logger.Errorf("Error Reading markdown files: %v", err)
		return nil, fmt.Errorf("Error Reading markdown files: %w", err)
	}

	todos := []*models.Todo{}

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
				t := &models.Todo{
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

	return todos, nil

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

type TemplateParameter struct {
	Name    string   `yaml:"name" json:"name"`
	Choices []string `yaml:"choices,omitempty" json:"choices,omitempty"`
	Blub    string   `json:"blub" yaml:"blub"`
}

type TemplateConfig struct {
	Name             string   `yaml:"name" json:"name"`
	Path             string   `yaml:"path" json:"path"`
	Parameters       []string `yaml:"parameters" json:"parameters"`
	OutputFolder     string   `yaml:"output_folder" json:"output_folder"`
	FilenameTemplate string   `yaml:"filename_template,omitempty" json:"filename_template,omitempty"`
}

type RenderTemplateInput struct {
	Template TemplateConfig    `json:"template" yaml:"template"`
	Options  map[string]string `json:"options" yaml:"options"`
}

func (g *Granite) RenderTemplate(args []string) (string, error) {
	g.logger.Debugf("Called RenderTemplate with args: %v", args)

	if len(args) != 2 {
		g.logger.Errorf("RenderTemplate expects 2 arguments, got: %d", len(args))
		return "", fmt.Errorf("RenderTemplate expects 2 arguments, got: %d", len(args))
	}

	templateConfig := TemplateConfig{}
	templateParameters := map[string]string{}

	err := json.Unmarshal([]byte(args[0]), &templateConfig)
	if err != nil {
		g.logger.Errorf("Cannot parse Arg 0 into template config: %v", err)
		return "", fmt.Errorf("Cannot parse Arg 0 into template config: %w", err)
	}

	err = json.Unmarshal([]byte(args[1]), &templateParameters)
	if err != nil {
		g.logger.Errorf("Cannot parse Arg 1 into options: %v", err)
		return "", fmt.Errorf("Cannot parse Arg 1 into options: %w", err)
	}

	// check input

	for _, v := range templateConfig.Parameters {
		if _, ok := templateParameters[v]; !ok {
			g.logger.Errorf("Template Parameter %s is undefined.", v)
			return "", fmt.Errorf("Template Parameter %s is undefined.", v)
		}
	}

	if templateConfig.FilenameTemplate == "" {
		if _, ok := templateParameters["filename"]; !ok {
			g.logger.Errorf("'filename' undefined in template parameters")
			return "", fmt.Errorf("'filename' undefined in template parameters")
		}
	} else {
		tpl, err := newTemplate("filename").Parse(templateConfig.FilenameTemplate)
		if err != nil {
			g.logger.Errorf("Could not parse filename template: %v", err)
			return "", fmt.Errorf("Could not parse filename template: %w", err)
		}
		b := strings.Builder{}
		err = tpl.Execute(&b, templateParameters)
		if err != nil {
			g.logger.Errorf("Error rendering filename template: %v", err)
			return "", fmt.Errorf("Error rendering filename template: %w", err)
		}
		templateParameters["filename"] = b.String()
	}

	templateContent, err := os.ReadFile(path.Join(g.RootPath, templateConfig.Path))
	if err != nil {
		log.Errorf("Error opening template file for reading: %v", err)
		return "", fmt.Errorf("Error opening template file for reading: %w", err)
	}

	tpl, err := newTemplate("file").Parse(string(templateContent))
	if err != nil {
		g.logger.Errorf("Could not parse template: %v", err)
		return "", fmt.Errorf("Could not parse template: %w", err)
	}

	b := strings.Builder{}
	err = tpl.Execute(&b, templateParameters)
	if err != nil {
		g.logger.Errorf("Error rendering template: %v", err)
		return "", fmt.Errorf("Error rendering template: %w", err)
	}
	outPath := path.Join(g.RootPath, templateConfig.OutputFolder, templateParameters["filename"])
	err = os.MkdirAll(filepath.Dir(outPath), 0755)
	if err != nil {
		g.logger.Errorf("Error creating folder holding file: %v", err)
		return "", fmt.Errorf("Error creating folder holding file: %w", err)
	}

	_, err = os.Stat(outPath)
	if errors.Is(err, os.ErrNotExist) {
		// write template cause file does not exist yet
		err = os.WriteFile(
			outPath,
			[]byte(b.String()),
			fs.ModePerm,
		)
	}

	return outPath, err
}

func (g *Granite) GetTemplates(args []string) (string, error) {
	b, err := json.Marshal(g.Templates)
	return string(b), err
}

type InitArgs struct {
	GraniteYaml string `json:"granite_yaml" yaml:"granite_yaml"`
	LogLevel    string `json:",omitempty" yaml:"log_level"`
}

type GraniteConfig struct {
	Templates []*TemplateConfig `json:"templates" yaml:"templates"`
	TodoTag   string            `json:"todotag" yaml:"todotag"`
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func (g *Granite) Init(args []string) (string, error) {
	g.logger.Infof("Called Init with args: %v", args)

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

	g.ConfigFile = initargs.GraniteYaml

	var logLevel log.Level = DEFAULT_LOGLEVEL
	if initargs.LogLevel != "" {
		logLevel, err = log.ParseLevel(initargs.LogLevel)
		if err != nil {
			log.Warnf("Couldn't parse loglevel %s: %v", initargs.LogLevel, err)
			return "", fmt.Errorf("Couldn't parse loglevel %s: %w", initargs.LogLevel, err)
		}
	}
	g.logger.SetLevel(logLevel)

	g.logger.Infof("Loading config file %s", g.ConfigFile)

	configRaw, err := os.ReadFile(g.ConfigFile)
	g.logger.Infof("Read file, checking err: %v", err)
	if err != nil {
		return "", err
	}

	graniteConf := &GraniteConfig{}
	err = yaml.Unmarshal(configRaw, graniteConf)
	g.logger.Infof("Unmarshalled conf, err: %v", err)
	if err != nil {
		g.logger.Errorf("Unable to parse config yaml: %v", err)
		return "", fmt.Errorf("Unable to parse config yaml: %w", err)
	}

	g.TodoTag = graniteConf.TodoTag
	g.Templates = graniteConf.Templates

	g.RootPath = filepath.Dir(g.ConfigFile)
	g.logger.Info("All Done in init")

	return "Called Init", nil
}

func (g *Granite) RunCodeblock(v *nvim.Nvim, args []string) {
	g.logger.Infof("Called RunCodeblock with args: %v", args)


	currentBuffer, err := v.CurrentBuffer()
	if err != nil {
		g.logger.Errorf("Unable to get current buffer: %v", err)
		return
	}

	currentWindow, err := v.CurrentWindow()
	if err != nil {
		g.logger.Errorf("Failed nvim call %v", err)
		return
	}

	lines, err := v.BufferLines(currentBuffer, 0, -1, false)
	if err != nil {
		g.logger.Errorf("Unable to get buffer lines: %v", err)
		return
	}

	cursorPosition, err := v.WindowCursor(currentWindow)
	if err != nil {
		g.logger.Errorf("Failed nvim call %v", err)
		return
	}

	sourceCode := bytes.Join(lines, []byte("\n"))

	codeblocks, err := GetCodeblocks(sourceCode)

	if err != nil {
		g.logger.Errorf("Error while parsing codeblocks: %v", err)
		return
	}

	var codeblockUnderCursor *codeblock.Codeblock
	for _, cb := range codeblocks {
		if cb.StartLine < cursorPosition[0] && cb.EndLine >= cursorPosition[0] {
			codeblockUnderCursor = cb
			break
		}
	}

	if codeblockUnderCursor == nil {
		return
	}

	if _, ok := codeblockUnderCursor.Opts["ID"]; !ok {
		codeblockUnderCursor.Opts["ID"] = fmt.Sprintf("%d", time.Now().UnixMilli())

		err = v.SetBufferLines(
			currentBuffer,
			codeblockUnderCursor.StartLine,
			codeblockUnderCursor.EndLine,
			false,
			codeblockUnderCursor.GetMarkdownLines(),
		)

		if err != nil {
			g.logger.Errorf("Couldn't set nvim bufferlines: %v", err)
			return
		}
	}

	clockGlyphs := []string{"󱑖", "󱑋", "󱑌", "󱑍", "󱑎", "󱑏", "󱑐", "󱑑", "󱑒", "󱑓", "󱑔", "󱑕"}
	checkmarkGlyph := ""
  
	if err != nil {
		g.logger.Errorf("unable to set extmark: %v", err)
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	signTickerDone := make(chan bool)
	defer func() {
		select{
    case signTickerDone <- true:
      return
    default:
      return
		}
    //v.DeleteBufferExtmark(currentBuffer, namespaceID, extmarkID)
	}()

	var targetCodeBlock *codeblock.Codeblock
	go func() {
		currentRun := 0
		for {
			select {
			case <-signTickerDone:
				return
			case <-ticker.C:
				curGlyph := clockGlyphs[currentRun%len(clockGlyphs)]
        SetExtmarkOnCodeblock(v, codeblockUnderCursor, curGlyph, "DiagnosticInfo")
        if targetCodeBlock != nil {
          SetExtmarkOnCodeblock(v, targetCodeBlock, curGlyph, "DiagnosticInfo")
        }
				currentRun++
			}
		}
	}()

	for _, cb := range codeblocks {
		id, ok := cb.Opts["SOURCE"]

		if ok && id == codeblockUnderCursor.Opts["ID"] {
			targetCodeBlock = cb
			break
		}
	}

	if targetCodeBlock == nil {
		v.SetBufferLines(currentBuffer, codeblockUnderCursor.EndLine+1, codeblockUnderCursor.EndLine+1, false, [][]byte{
			[]byte("```out SOURCE=" + codeblockUnderCursor.Opts["ID"]),
			[]byte(""),
			[]byte("```"),
			[]byte(""),
		})

		lines, err = v.BufferLines(currentBuffer, 0, -1, false)
		if err != nil {
			g.logger.Errorf("Unable to get buffer lines: %v", err)
			return
		}
		sourceCode = bytes.Join(lines, []byte("\n"))

		codeblocks, err = GetCodeblocks(sourceCode)
		if err != nil {
			g.logger.Errorf("Error while parsing codeblocks: %v", err)
			return
		}

		for _, cb := range codeblocks {
			id, ok := cb.Opts["SOURCE"]

			if ok && id == codeblockUnderCursor.Opts["ID"] {
				targetCodeBlock = cb
				break
			}
		}
	}

	if targetCodeBlock == nil {
		g.logger.Error("Output codeblock wasnt found")
		return
	}

	envVars := codeblockUnderCursor.PopulateOpts(sourceCode)
	command, err := codeblock.GetCommandForCodeblock(codeblockUnderCursor, envVars)
	if err != nil {
		g.logger.Errorf("Error Creating command: %v", err)
		return
	}

	outbytes, err := command.CombinedOutput()
	signTickerDone <- true
  errorGlyph := "󱂑"
  var outGlyph string
  var outHighlight string
	if err != nil {
    outGlyph = errorGlyph
    outHighlight = "DiagnosticError"
	} else {
    outGlyph = checkmarkGlyph
    outHighlight = "DiagnosticOk"
	}

	targetCodeBlock.Text = string(outbytes)
	targetCodeBlock.Opts[codeblock.CB_OPT_LAST_RUN] = time.Now().Format(time.RFC3339)
	err = v.SetBufferLines(
		currentBuffer,
		targetCodeBlock.StartLine,
		targetCodeBlock.EndLine,
		false,
		targetCodeBlock.GetMarkdownLines(),
	)

  SetExtmarkOnCodeblock(v, codeblockUnderCursor, outGlyph, outHighlight)
  SetExtmarkOnCodeblock(v, targetCodeBlock, outGlyph, outHighlight)

	if err != nil {
		g.logger.Errorf("Error writing output: %v", err)
	}
}

func SetExtmarkOnCodeblock(v *nvim.Nvim, cb *codeblock.Codeblock, text string, hlgroup string) error {
  currentBuffer, err := v.CurrentBuffer()
  if err != nil {
    return err
  }

	namespaceID, err := v.CreateNamespace(EXTMARK_NS)
	if err != nil {
		return err
	}
  var extmarkID int 
  if cb.Opts[codeblock.CB_OPT_ID] != "" {
    extmarkID, err = strconv.Atoi(cb.Opts[codeblock.CB_OPT_ID])
  } else {
    extmarkID, err = strconv.Atoi(cb.Opts[codeblock.CB_OPT_SOURCE])
    extmarkID ++
  }

	if err != nil {
		return err
	}

  _, err = v.SetBufferExtmark(currentBuffer, namespaceID, cb.StartLine, 0, map[string]interface{}{
    "id":        extmarkID,
    "virt_text": [][]interface{}{{text, hlgroup}},
  })
  
  return err
}

func GetCodeblocks(sourceCode []byte) ([]*codeblock.Codeblock, error) {
	tsparser := ts.NewParser()
	tsparser.SetLanguage(markdown.GetLanguage())
	tree, err := tsparser.ParseCtx(context.TODO(), nil, sourceCode)

	if err != nil {
		return nil, err
	}

	fencedCodeBlocksPatern := "(fenced_code_block) @cb"
	query, err := ts.NewQuery([]byte(fencedCodeBlocksPatern), markdown.GetLanguage())
	if err != nil {
		return nil, err
	}
	qc := ts.NewQueryCursor()

	qc.Exec(query, tree.RootNode())

	codeblockNodes := []*ts.Node{}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, c := range m.Captures {
			codeblockNodes = append(codeblockNodes, c.Node)
		}
	}

	codeBlocks := []*codeblock.Codeblock{}

	for _, n := range codeblockNodes {
		cb, err := codeblock.NewCodeblockFromNode(n, sourceCode)
		if err != nil {
			continue
		}
		codeBlocks = append(codeBlocks, cb)
	}
	return codeBlocks, nil
}

func main() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	logfile := path.Join(homedir, ".granite_go.log")
	f, err := os.OpenFile(
		logfile,
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
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteRunCodeblock"}, g.RunCodeblock)
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetTodos"}, g.GetTodos)
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetAllTags"}, g.GetAllTags)
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetTemplates"}, g.GetTemplates)
		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteRenderTemplate"}, g.RenderTemplate)
		p.HandleFunction(&plugin.FunctionOptions{
			Name: "GraniteInit",
		}, g.Init)
		return nil
	})
	g.logger.Infof("after plugin init")
}
