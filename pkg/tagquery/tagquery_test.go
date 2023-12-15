package tagquery

import (
	"reflect"
	"testing"

	"github.com/mrWinston/granite.nvim/pkg/models"
)

func TestTodosOR(t *testing.T) {
	type args struct {
		lhs []*models.Todo
		rhs []*models.Todo
	}
	tests := []struct {
		name string
		args args
		want []*models.Todo
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TodosOR(tt.args.lhs, tt.args.rhs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TodosOR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTodosAND(t *testing.T) {
	type args struct {
		lhs []*models.Todo
		rhs []*models.Todo
	}
	tests := []struct {
		name string
		args args
		want []*models.Todo
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TodosAND(tt.args.lhs, tt.args.rhs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TodosAND() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryToTokens(t *testing.T) {
	tests := []struct {
		name string
		args string
		want []string
	}{
		{
			name: "one",
			args: "((blabli AND blob) OR blib )sdflkj ksdjfk",
			want: []string{"(", "(", "blabli", "AND", "blob", ")", "OR", "blib", ")", "sdflkj", "ksdjfk"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QueryToTokens(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryToTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareTrees(t1 *TagQueryToken, t2 *TagQueryToken) bool {

	// both nil ,return true
	if (t1 == nil) && (t2 == nil) {
		return true
	}
	// only one is nil, return false
	if (t1 == nil) != (t2 == nil) {
		return false
	}

	// content doesn't match
	if t1.Content != t2.Content {
		return false
	}

	// check if children are equal
	return compareTrees(t1.Lhs, t2.Lhs) && compareTrees(t1.Rhs, t2.Rhs)
}

func TestBuildTokenTree(t *testing.T) {
	// t1
	q1 := []string{"hello", "AND", "olleh"}

	t11 := &TagQueryToken{
		Content: "hello",
	}
	t12 := &TagQueryToken{
		Content: "AND",
	}
	t13 := &TagQueryToken{
		Content: "olleh",
	}

	t12.Lhs = t11
	t12.Lhs.Parent = t12
	t12.Rhs = t13
	t12.Rhs.Parent = t12

	r1, err := BuildTokenTree(q1)
	if err != nil {
		t.Errorf("Didn't expect an error in T1: %v", err)
	}
	if !reflect.DeepEqual(r1, t12) {
		t.Errorf("Simple Test does not match: want = %v, got = %v", t12, r1)
	}

	// t2

	q2 := []string{"(", "aaa", "OR", "bbb", ")", "AND", "(", "ccc", "AND", "ddd", ")"}

	tr := NewToken("AND")
	tr.SetLhs(NewToken("OR"))
	tr.SetRhs(NewToken("AND"))
	tr.Lhs.SetLhs(NewToken("aaa"))
	tr.Lhs.SetRhs(NewToken("bbb"))
	tr.Rhs.SetLhs(NewToken("ccc"))
	tr.Rhs.SetRhs(NewToken("ddd"))

	r2, err := BuildTokenTree(q2)
	if err != nil {
		t.Errorf("Didn't expect an error in T2: %v", err)
	}
	if !reflect.DeepEqual(r2, tr) {
		t.Errorf("Simple Test does not match: want = %v, got = %v", tr, r2)
	}

	// test dumb tree

	q3 := []string{"lskajdf", "lsjdfksdf", "ksjkdfj"}
	_, err = BuildTokenTree(q3)

	if err != INVALID_QUERY_ERROR {
		t.Errorf("Expected %v but received: %v", INVALID_QUERY_ERROR, err)
	}
	q4 := []string{"(", "a", "AND", "b"}
	_, err = BuildTokenTree(q4)

	if err != INVALID_QUERY_ERROR {
		t.Errorf("Expected an Error in T4, but didn't receive one")
	}
}

func TestTagQueryToken_SetLhs(t *testing.T) {
	type fields struct {
		Parent  *TagQueryToken
		Content string
		Lhs     *TagQueryToken
		Rhs     *TagQueryToken
		Todos   []*models.Todo
	}
	type args struct {
		lhs *TagQueryToken
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TagQueryToken{
				Parent:  tt.fields.Parent,
				Content: tt.fields.Content,
				Lhs:     tt.fields.Lhs,
				Rhs:     tt.fields.Rhs,
				Todos:   tt.fields.Todos,
			}
			tr.SetLhs(tt.args.lhs)
		})
	}
}

func TestTagQueryToken_SetRhs(t *testing.T) {
	type fields struct {
		Parent  *TagQueryToken
		Content string
		Lhs     *TagQueryToken
		Rhs     *TagQueryToken
		Todos   []*models.Todo
	}
	type args struct {
		rhs *TagQueryToken
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TagQueryToken{
				Parent:  tt.fields.Parent,
				Content: tt.fields.Content,
				Lhs:     tt.fields.Lhs,
				Rhs:     tt.fields.Rhs,
				Todos:   tt.fields.Todos,
			}
			tr.SetRhs(tt.args.rhs)
		})
	}
}

func TestTagQueryToken_String(t *testing.T) {
	type fields struct {
		Parent  *TagQueryToken
		Content string
		Lhs     *TagQueryToken
		Rhs     *TagQueryToken
		Todos   []*models.Todo
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TagQueryToken{
				Parent:  tt.fields.Parent,
				Content: tt.fields.Content,
				Lhs:     tt.fields.Lhs,
				Rhs:     tt.fields.Rhs,
				Todos:   tt.fields.Todos,
			}
			if got := tr.String(); got != tt.want {
				t.Errorf("TagQueryToken.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewToken(t *testing.T) {
	type args struct {
		c string
	}
	tests := []struct {
		name string
		args args
		want *TagQueryToken
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewToken(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagQueryToken_GetTodos(t *testing.T) {

}
