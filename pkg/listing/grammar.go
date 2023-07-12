package listing

import (
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = strings.ToLower(values[0]) == "true"
	return nil
}

// nolint: govet
type Expression struct {
	And []ConditionGrammar `( WS? @@ )*`
}

// nolint: govet
type ConditionGrammar struct {
	Not     bool     `@( "NOT" WS )?`
	Symbol  string   `@Identifier @( "." Identifier )*`
	Compare *Compare `(   WS? @@`
	Between *Between `  | WS? ":" WS? "[" WS? @@ WS? "]"`
	In      *In      `  | WS "IN" WS? "(" WS? @@ ")" )`
}

// nolint: govet
type Compare struct {
	Operator string `@Operator WS?`
	Value    Value  `  @@ WS?`
}

// nolint: govet
type Between struct {
	Start Value `@@ WS?`
	End   Value `"," WS? @@`
}

// nolint: govet
type In struct {
	Values []Value `@@ WS? ( "," WS? @@ WS? )*`
}

// nolint: govet
type Value struct {
	Int     *int64   `  @Int`
	Float   *float64 `| @Float`
	String  *string  `| @String`
	Boolean *Boolean `| @("TRUE" | "FALSE")`
}

var parser = participle.MustBuild(
	&Expression{},
	participle.Lexer(lexer.Must(lexer.Regexp(`(?P<WS>\s+)`+
		`|(?P<Keyword>(?i)\b(?:NOT|IN|TRUE|FALSE)\b)`+
		`|(?P<Identifier>[a-zA-Z_][a-zA-Z0-9_]*)`+
		`|(?P<Float>[-+]?\d*\.\d+([eE][-+]?\d+)?)`+
		`|(?P<Int>[-+]?\d+([eE][-+]?\d+)?)`+
		`|(?P<String>'[^']*'|"[^"]*")`+
		`|(?P<Separator>[,\.])`+
		`|(?P<Bracket>[\[\]\(\)])`+
		`|(?P<Operator>!=|<=|>=|[:=<>])`,
	))),
	participle.Unquote("String"),
	participle.CaseInsensitive("Keyword"),
	// Lookahead needed to disambiguate contains (field: "val") and range (field: [0, 1])
	participle.UseLookahead(2),
)

func ParseGrammar(input string, expr *Expression) error {
	return parser.ParseString(input, expr)
}
