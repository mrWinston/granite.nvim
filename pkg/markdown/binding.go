package markdown

//#include "tree_sitter/parser.h"
//TSLanguage *tree_sitter_markdown();
import "C"
import (
	"unsafe"

	ts "github.com/smacker/go-tree-sitter"
)

func GetLanguage() *ts.Language {
	ptr := unsafe.Pointer(C.tree_sitter_markdown())
	return ts.NewLanguage(ptr)
}
