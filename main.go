package main

import (
	"log"
	"os"
	"strings"

	"github.com/neovim/go-client/nvim/plugin"
)

type Granite struct {
	RootPath           string
	TemplateConfigPath string
	TodoTag            string
}

func (g *Granite) GetTodos(args []string) (string, error) {
	return "", nil
}

func GetTodos(args []string) (string, error) {
	return "Hello " + strings.Join(args, " "), nil
}

func main() {
  
	l, _ := os.Create("nvim-go-client-example.log")

	log.SetOutput(l)
  
	plugin.Main(func(p *plugin.Plugin) error {
		p.HandleFunction(&plugin.FunctionOptions{Name: "GetTodos"}, GetTodos)
		return nil
	})
}

// 1. init with settings
