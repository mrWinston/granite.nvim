package tagquery

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mrWinston/granite.nvim/pkg/models"
)

func TodosOR(lhs []*models.Todo, rhs []*models.Todo) []*models.Todo {
	out := []*models.Todo{}
	out = append(out, lhs...)
	return append(out, rhs...)
}

func TodosAND(lhs []*models.Todo, rhs []*models.Todo) []*models.Todo {
	out := []*models.Todo{}

	for _, l := range lhs {
		for _, r := range rhs {
			if l == r {
				out = append(out, l)
			}
		}
	}
	return out
}

func QueryToTokens(query string) []string {
	out := []string{}
	buffer := ""
	endTokenChars := "() "
	for _, currentChar := range query {
		if strings.Contains(endTokenChars, string(currentChar)) {
			if buffer != "" {
				out = append(out, buffer)
				buffer = ""
			}

			if currentChar == ' ' {
				continue
			}
			out = append(out, string(currentChar))
		} else {
			buffer = buffer + string(currentChar)
		}
	}

	if buffer != "" {
		out = append(out, buffer)
	}

	return out
}

type TagQueryToken struct {
	Parent  *TagQueryToken
	Content string
	Lhs     *TagQueryToken
	Rhs     *TagQueryToken
	Todos   []*models.Todo
}

func (t *TagQueryToken) SetLhs(lhs *TagQueryToken) {
	t.Lhs = lhs
	lhs.Parent = t
}

func (t *TagQueryToken) SetRhs(rhs *TagQueryToken) {
	t.Rhs = rhs
	rhs.Parent = t
}

func (t TagQueryToken) String() string {
	return fmt.Sprintf("{ Content: '%s', Lhs: %s, Rhs: %s }", t.Content, t.Lhs, t.Rhs)
}

func NewToken(c string) *TagQueryToken {
	return &TagQueryToken{
		Content: c,
	}
}

var INVALID_QUERY_ERROR error = errors.New("Could not parse Query")

func BuildTokenTree(rawTokens []string) (*TagQueryToken, error) {
	root := &TagQueryToken{}

	currentToken := &TagQueryToken{
		Parent: root,
	}
	root.Lhs = currentToken
	for _, rawToken := range rawTokens {

		if currentToken == nil {
			return nil, INVALID_QUERY_ERROR
		}

		switch rawToken {
		case "(":
			currentToken.SetLhs(NewToken(""))
			currentToken = currentToken.Lhs
		case ")":
			currentToken = currentToken.Parent
		case "AND", "OR":
			currentToken.Content = rawToken
			currentToken.SetRhs(NewToken(""))
			currentToken = currentToken.Rhs
		default:
			currentToken.Content = rawToken
			currentToken = currentToken.Parent
		}
	}
	if currentToken != root {
		return nil, INVALID_QUERY_ERROR
	}

  for currentToken.Content == "" && currentToken.Rhs == nil {
    currentToken = currentToken.Lhs
  }


	return currentToken, nil
}

func (t *TagQueryToken) GetTodos(filter func(string) []*models.Todo) []*models.Todo {
  if t.Content == "AND" {
    return TodosAND(t.Lhs.GetTodos(filter), t.Rhs.GetTodos(filter))
  }
  if t.Content == "OR" {
    return TodosOR(t.Lhs.GetTodos(filter), t.Rhs.GetTodos(filter))
  }
  return filter(t.Content)
}
