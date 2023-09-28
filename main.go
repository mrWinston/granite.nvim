package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/neovim/go-client/nvim/plugin"
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
  Text string
  // Tags are the tags assigned to the todo
  Tags []string
  // LineNumber is the line in the file where the todo was found
  LineNumber int
  // FilePath is the path to the file where the todo was found
  FilePath string
  // DueDate is the date when the todo is due. The format must be YYYY-MM-DD
  DueDate string
  // StateString is a string representation of the State of the ToDo, eg: OPEN, DONE, IN_PROGRESS
  StateString string
}

// Parse parses the RawLine set in the todo and populates all other fields based on what it finds there.
//
// Returns an error when RawLine, LineNumber or FilePath are unset or the RawLine can't be parsed into a todo
func (t *Todo) Parse() error {
  
}


func (g *Granite) GetTodos(args []string) (string, error) {
	g.logger.Printf("Called GetTodos with args: %v\n", args)
  mdFiles, err := GetAllFilesWithExtInDir(g.RootPath, ".md")

  if err != nil {
    return "", err
  }
   
  for _, mdFilePath := range mdFiles {
    mdFile, err := os.Open(mdFilePath) 
    if err != nil {
      g.logger.Printf("Error opening file %s for reading: %v. Skipping.", mdFilePath, err)
      continue
    }   
    scanner := bufio.NewScanner(mdFile)
    scanner.Split(bufio.ScanLines)
    lineNumber := 1
    if ! scanner.Scan() {
      g.logger.Printf("Scanning finished prematurely: %s", mdFilePath)
      continue
    }
    for line := scanner.Text(); scanner.Scan(); line = scanner.Text() {
      if strings.Contains(g.TodoTag) {
         
      }
      lineNumber++
    }
  }
	return "Called GetTodos", nil
}

func GetAllFilesWithExtInDir(dir string, ext string) ([]string, error) {
	foundFiles := []string{}
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

      if info.IsDir(){
        return nil
      }

      if ! strings.HasSuffix(path, ext) {
        return nil
      }
      foundFiles = append(foundFiles, path)
			return nil
		})

	return foundFiles, err
}

func GetTodos(args []string) (string, error) {
	return "Hello " + strings.Join(args, " "), nil
}

type InitArgs struct {
	RootPath           string
	TemplateConfigPath string
	TodoTag            string
}
func (g *Granite) Init(args []string) (string, error) { g.logger.Printf("Called Init with args: %v\n", args) if len(args) != 1 { g.logger.Printf("Init expects 1 argument, got: %d", len(args))
		return "", fmt.Errorf("Init expects 1 argument, got: %d", len(args))
	}

	initargs := &InitArgs{}

	err := json.Unmarshal([]byte(args[0]), initargs)
	if err != nil {
		g.logger.Printf("Couln't parse json data: %v", err)
		return "", fmt.Errorf("Couln't parse json data: %w", err)
	}

	g.TodoTag = initargs.TodoTag
	g.TemplateConfigPath = initargs.TemplateConfigPath
	g.RootPath = initargs.RootPath

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
	defer f.Close()
	g := &Granite{
		logger: log.New(f, "granite", log.Flags()),
	}

	g.logger.Println("Logger init done")
	plugin.Main(func(p *plugin.Plugin) error {

		p.HandleFunction(&plugin.FunctionOptions{Name: "GraniteGetTodos"}, g.GetTodos)
		p.HandleFunction(&plugin.FunctionOptions{
			Name: "GraniteInit",
		}, g.Init)
		return nil
	})
	g.logger.Println("after plugin init")
}

// 1. init with settings
