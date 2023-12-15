package codeblock

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ts "github.com/smacker/go-tree-sitter"
)

type Codeblock struct {
	Language  string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
	Node      *ts.Node
	Opts      map[string]string
	Text      string
}

const (
	CB_OPT_WORKDIR               = "CWD"
	CB_OPT_WORKDIR_DOCKER_PREFIX = "docker"
	CB_OPT_ID                    = "ID"
	CB_OPT_SOURCE                = "SOURCE"
	CB_OPT_LAST_RUN              = "LAST_RUN"
)

var (
	NotACodeblockError = errors.New("Node is not a fenced codeblock")
	DefaultShell       = "zsh"
)

func NewCodeblockFromNode(node *ts.Node, sourcecode []byte) (*Codeblock, error) {

	if node.Type() != "fenced_code_block" {
		return nil, NotACodeblockError
	}

	cb := &Codeblock{
		Node:      node,
		StartLine: int(node.StartPoint().Row),
		EndLine:   int(node.EndPoint().Row),
		StartCol:  int(node.StartPoint().Column),
		EndCol:    int(node.EndPoint().Column),
		Text:      node.Content(sourcecode),
		Opts:      make(map[string]string),
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		currentChild := node.Child(i)
		if currentChild == nil {
			break
		}

		if currentChild.Type() == "info_string" {
			infoString := currentChild.Content(sourcecode)
			infoSplit := strings.Split(infoString, " ")
			if len(infoSplit) == 0 {
				continue
			}
			cb.Language = infoSplit[0]

			if len(infoSplit) == 1 {
				continue
			}

			for _, v := range infoSplit[1:] {
				keyValSplit := strings.Split(v, "=")
				if len(keyValSplit) != 2 {
					continue
				}
				cb.Opts[keyValSplit[0]] = keyValSplit[1]
			}

		} else if currentChild.Type() == "code_fence_content" {
			cb.Text = currentChild.Content(sourcecode)
		}
	}

	return cb, nil
}

func (cb *Codeblock) PopulateOpts(sourcecode []byte) map[string]string {
	envMap := map[string]string{}
	currentNode := cb.Node.Parent()

	for currentNode != nil && currentNode.Type() == "section" {
		childNodes := getChildNodesWithType(currentNode, "fenced_code_block")
		for _, childNode := range childNodes {
			childCb, err := NewCodeblockFromNode(childNode, sourcecode)
			if err != nil {
				continue
			}
			if childCb.Language != "env" {
				continue
			}

			for _, line := range strings.Split(childCb.Text, "\n") {
				kvSplit := strings.Split(line, "=")
				if len(kvSplit) != 2 {
					continue
				}

				key := strings.Trim(kvSplit[0], " ")
				val := strings.Trim(kvSplit[1], " ")

				if _, ok := envMap[key]; !ok {
					envMap[key] = val
				}
			}

		}
		currentNode = currentNode.Parent()
	}

	return envMap
}

func (cb *Codeblock) GetMarkdownLines() [][]byte {
	var sb strings.Builder

	newLines := [][]byte{}

	sb.WriteString("```")
	sb.WriteString(cb.Language)

	for key, val := range cb.Opts {
		sb.WriteString(" ")
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(val)
	}
	newLines = append(newLines, []byte(sb.String()))
	sb.Reset()
	for _, line := range strings.Split(cb.Text, "\n") {
		newLines = append(newLines, []byte(line))
	}
	newLines = newLines[:len(newLines)-1]
	newLines = append(newLines, []byte("```"))
	newLines = append(newLines, []byte(""))

	return newLines
}

func GetCommandForCodeblock(codeblock *Codeblock, envVars map[string]string) (*exec.Cmd, error) {
	var outCommand *exec.Cmd
	var err error
	switch codeblock.Language {
	case "sh", "zsh", "bash":
		outCommand, err = getShellCommand(codeblock, envVars)
	default:
		return nil, fmt.Errorf("Language %s can't be executed", codeblock.Language)
	}

	return outCommand, err
}

func getShellCommand(codeblock *Codeblock, envVars map[string]string) (*exec.Cmd, error) {
	var outCommand *exec.Cmd
	var err error

	tmpfile, err := os.CreateTemp(os.TempDir(), "granite.tmpfile")

	if err != nil {
		return nil, err
	}

	_, err = tmpfile.WriteString(codeblock.Text)
	if err != nil {
		return nil, err
	}

	var shellCmd string
	if strings.HasPrefix(codeblock.Opts[CB_OPT_WORKDIR], CB_OPT_WORKDIR_DOCKER_PREFIX) {
		cwdSplit := strings.Split(codeblock.Opts[CB_OPT_WORKDIR], ":")
		if len(cwdSplit) != 2 {
			return nil, fmt.Errorf("Malformed cwd entry")
		}

		var sb strings.Builder
		for k, v := range envVars {
			sb.WriteString(fmt.Sprintf(" -e '%s=%s'", k, v))
		}

		containerName := cwdSplit[1]
		if envVars[containerName] != "" {
			containerName = envVars[containerName]
		}
		shellCmd = fmt.Sprintf(
			"podman cp %[1]s %[2]s:/tmpscript && podman exec -i %[3]s %[2]s bash -c 'source /tmpscript'",
			tmpfile.Name(),
			containerName,
			sb.String(),
		)
	} else {
		shellCmd = fmt.Sprintf(
			"source %s",
			tmpfile.Name(),
		)

	}

	outCommand = exec.Command(DefaultShell, "-i", "-c", shellCmd)
  outCommand.Env = append(outCommand.Env, os.Environ()...)
	for k, v := range envVars {
		outCommand.Env = append(outCommand.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return outCommand, err

}

func getChildNodesWithType(node *ts.Node, nodeType string) []*ts.Node {

	children := []*ts.Node{}
	for i := 0; i < int(node.ChildCount()); i++ {
		currentChild := node.Child(i)
		if currentChild == nil || currentChild.Type() != nodeType {
			continue
		}
		children = append(children, currentChild)
	}

	return children
}
