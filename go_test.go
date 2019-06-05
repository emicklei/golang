package golang

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compareBytes is a helper for comparing expected output to generated output
func compareBytes(t *testing.T, ex, out []byte) {
	if bytes.EqualFold(out, ex) {
		return
	}
	fmt.Println(string(out))

	line := 1
	for i, b := range out {
		if b == '\n' {
			line++
		}

		if ex[i] != b {
			t.Fatalf("expected: %s, but got: %s, %d:%d", string(ex[i]), string(b), i, line)
		}
	}
}

var (
	// Flags are used here to allow for the input/output files to be changed during dev
	// One use case of changing the files is to examine how Generate scales through the benchmark
	//
	gqlFileName = flag.String("gqlFile", "test.gql", "Specify a .gql file to use a input for testing.")
	exDocName   = flag.String("expectedFile", "test.gotxt", "Specify a file which is the expected generator output from the given .gql file.")

	testDoc *ast.Document
	exDoc   io.Reader
)

func TestMain(m *testing.M) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Parse flags
	flag.Parse()

	// Assume the input file is in the current working directory
	if !filepath.IsAbs(*gqlFileName) {
		*gqlFileName = filepath.Join(wd, *gqlFileName)
	}
	f, err := os.Open(*gqlFileName)
	if err != nil {
		panic(err)
	}

	// Assume the output file is in the current working directory
	if !filepath.IsAbs(*exDocName) {
		*exDocName = filepath.Join(wd, *exDocName)
	}
	exDoc, err = os.Open(*exDocName)
	if err != nil {
		panic(err)
	}

	testDoc, err = parser.ParseDoc(token.NewDocSet(), "test", f, 0)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestScalar(t *testing.T) {
	g := &Generator{}

	ts := &ast.TypeSpec{
		Name: &ast.Ident{Name: "Test"},
	}

	g.generateScalar("Test", false, nil, ts)

	ex := []byte(`NewScalar(graphql.ScalarConfig{
	Name: "Test",
	Serialize: func(value interface{}) interface{} { return nil }, // TODO
})
`)

	compareBytes(t, ex, g.Bytes())
}

func TestObject(t *testing.T) {
	g := &Generator{}

	t.Run("JustFields", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Object{
			Object: &ast.ObjectType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Name: &ast.Ident{Name: "one"},
							Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "Int"}},
						},
						{
							Name: &ast.Ident{Name: "str"},
							Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "String"}},
						},
						{
							Name: &ast.Ident{Name: "list"},
							Type: &ast.Field_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Test"}}}},
						},
					},
				},
			},
		}}

		g.generateObject("Test", false, nil, ts)

		ex := []byte(`NewObject(graphql.ObjectConfig{
	Name: "Test",
	Fields: graphql.Fields{
		"one": &graphql.Field{
			Type: graphql.Int,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
		"str": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
		"list": &graphql.Field{
			Type: graphql.NewList(TestType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})

	t.Run("WithInterfaces", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Object{
			Object: &ast.ObjectType{
				Interfaces: []*ast.Ident{{Name: "A"}, {Name: "B"}},
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Name: &ast.Ident{Name: "one"},
							Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "Int"}},
						},
						{
							Name: &ast.Ident{Name: "str"},
							Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "String"}},
						},
						{
							Name: &ast.Ident{Name: "list"},
							Type: &ast.Field_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Test"}}}},
						},
					},
				},
			},
		}}

		g.generateObject("Test", false, nil, ts)

		ex := []byte(`NewObject(graphql.ObjectConfig{
	Name: "Test",
	Interfaces: []*graphql.Interface{
		AType,
		BType,
	},
	Fields: graphql.Fields{
		"one": &graphql.Field{
			Type: graphql.Int,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
		"str": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
		"list": &graphql.Field{
			Type: graphql.NewList(TestType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
		},
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})
}

func TestInterface(t *testing.T) {
	g := &Generator{}

	ts := &ast.TypeSpec{Type: &ast.TypeSpec_Interface{
		Interface: &ast.InterfaceType{
			Fields: &ast.FieldList{
				List: []*ast.Field{
					{
						Name: &ast.Ident{Name: "one"},
						Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "Int"}},
					},
					{
						Name: &ast.Ident{Name: "str"},
						Type: &ast.Field_Ident{Ident: &ast.Ident{Name: "String"}},
					},
					{
						Name: &ast.Ident{Name: "list"},
						Type: &ast.Field_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Test"}}}},
					},
				},
			},
		},
	}}

	g.generateInterface("Test", false, nil, ts)

	ex := []byte(`NewInterface(graphql.InterfaceConfig{
	Name: "Test",
	Fields: graphql.Fields{
		"one": &graphql.Field{
			Type: graphql.Int,
		},
		"str": &graphql.Field{
			Type: graphql.String,
		},
		"list": &graphql.Field{
			Type: graphql.NewList(TestType),
		},
	},
})
`)

	compareBytes(t, ex, g.Bytes())
}

func TestUnion(t *testing.T) {
	g := &Generator{}

	ts := &ast.TypeSpec{Type: &ast.TypeSpec_Union{
		Union: &ast.UnionType{
			Members: []*ast.Ident{{Name: "A"}, {Name: "B"}},
		},
	}}

	g.generateUnion("Test", false, nil, ts)

	ex := []byte(`NewUnion(graphql.UnionConfig{
	Name: "Test",
	Types: []*graphql.Object{
		AType,
		BType,
	},
	ResolveType: func(p graphql.ResolveParams) *graphql.Object { return nil }, // TODO
})
`)

	compareBytes(t, ex, g.Bytes())
}

func TestEnum(t *testing.T) {
	g := &Generator{}

	ts := &ast.TypeSpec{Type: &ast.TypeSpec_Enum{
		Enum: &ast.EnumType{
			Values: &ast.FieldList{
				List: []*ast.Field{
					{Name: &ast.Ident{Name: "A"}},
					{Name: &ast.Ident{Name: "B"}},
					{Name: &ast.Ident{Name: "C"}},
				},
			},
		},
	}}

	g.generateEnum("Test", false, nil, ts)

	ex := []byte(`NewEnum(graphql.EnumConfig{
	Name: "Test",
	Values: graphql.EnumValueConfigMap{
		"A": &graphql.EnumValueConfig{
			Value: "A",
		},
		"B": &graphql.EnumValueConfig{
			Value: "B",
		},
		"C": &graphql.EnumValueConfig{
			Value: "C",
		},
	},
})
`)

	compareBytes(t, ex, g.Bytes())
}

func TestInput(t *testing.T) {
	g := &Generator{}

	t.Run("NoDefaults", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Input{
			Input: &ast.InputType{
				Fields: &ast.InputValueList{
					List: []*ast.InputValue{
						{
							Name: &ast.Ident{Name: "one"},
							Type: &ast.InputValue_Ident{Ident: &ast.Ident{Name: "Int"}},
						},
						{
							Name: &ast.Ident{Name: "str"},
							Type: &ast.InputValue_Ident{Ident: &ast.Ident{Name: "String"}},
						},
						{
							Name: &ast.Ident{Name: "list"},
							Type: &ast.InputValue_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Test"}}}},
						},
					},
				},
			},
		}}

		g.generateInput("Test", false, nil, ts)

		ex := []byte(`NewInputObject(graphql.InputObjectConfig{
	Name: "Test",
	Fields: graphql.InputObjectFieldConfigMap{
		"one": &graphql.InputObjectFieldConfig{
			Type: graphql.Int,
		},
		"str": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"list": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(TestType),
		},
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})

	t.Run("WithDefaults", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Input{
			Input: &ast.InputType{
				Fields: &ast.InputValueList{
					List: []*ast.InputValue{
						{
							Name: &ast.Ident{Name: "one"},
							Type: &ast.InputValue_Ident{Ident: &ast.Ident{Name: "Int"}},
							Default: &ast.InputValue_BasicLit{BasicLit: &ast.BasicLit{
								Kind:  token.Token_INT,
								Value: "1",
							}},
						},
						{
							Name: &ast.Ident{Name: "str"},
							Type: &ast.InputValue_Ident{Ident: &ast.Ident{Name: "String"}},
							Default: &ast.InputValue_BasicLit{BasicLit: &ast.BasicLit{
								Kind:  token.Token_STRING,
								Value: `"hello"`,
							}},
						},
						{
							Name: &ast.Ident{Name: "list"},
							Type: &ast.InputValue_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Int"}}}},
							Default: &ast.InputValue_CompositeLit{CompositeLit: &ast.CompositeLit{
								Value: &ast.CompositeLit_ListLit{
									ListLit: &ast.ListLit{
										List: &ast.ListLit_BasicList{
											BasicList: &ast.ListLit_Basic{
												Values: []*ast.BasicLit{
													{Kind: token.Token_INT, Value: "1"},
													{Kind: token.Token_INT, Value: "2"},
													{Kind: token.Token_INT, Value: "3"},
												},
											},
										},
									},
								},
							}},
						},
					},
				},
			},
		}}

		g.generateInput("Test", false, nil, ts)

		ex := []byte(`NewInputObject(graphql.InputObjectConfig{
	Name: "Test",
	Fields: graphql.InputObjectFieldConfigMap{
		"one": &graphql.InputObjectFieldConfig{
			Type: graphql.Int,
			DefaultValue: 1,
		},
		"str": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
			DefaultValue: "hello",
		},
		"list": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(graphql.Int),
			DefaultValue: []interface{}{1, 2, 3},
		},
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})
}

func TestDirective(t *testing.T) {
	g := &Generator{}

	t.Run("NoArgs", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Directive{
			Directive: &ast.DirectiveType{
				Locs: []*ast.DirectiveLocation{
					{Loc: ast.DirectiveLocation_QUERY},
					{Loc: ast.DirectiveLocation_FIELD},
					{Loc: ast.DirectiveLocation_SCHEMA},
				},
			},
		}}

		g.generateDirective("Test", false, nil, ts)

		ex := []byte(`NewDirective(graphql.DirectiveConfig{
	Name: "Test",
	Locations: []string{
		"QUERY",
		"FIELD",
		"SCHEMA",
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})

	t.Run("WithArgs", func(subT *testing.T) {
		g.Lock()
		defer g.Unlock()
		g.Reset()

		ts := &ast.TypeSpec{Type: &ast.TypeSpec_Directive{
			Directive: &ast.DirectiveType{
				Locs: []*ast.DirectiveLocation{
					{Loc: ast.DirectiveLocation_QUERY},
					{Loc: ast.DirectiveLocation_FIELD},
					{Loc: ast.DirectiveLocation_SCHEMA},
				},
				Args: &ast.InputValueList{
					List: []*ast.InputValue{
						{
							Name:    &ast.Ident{Name: "one"},
							Type:    &ast.InputValue_Ident{Ident: &ast.Ident{Name: "Int"}},
							Default: &ast.InputValue_BasicLit{BasicLit: &ast.BasicLit{Value: "1"}},
						},
						{
							Name:    &ast.Ident{Name: "str"},
							Type:    &ast.InputValue_NonNull{NonNull: &ast.NonNull{Type: &ast.NonNull_Ident{Ident: &ast.Ident{Name: "String"}}}},
							Default: &ast.InputValue_BasicLit{BasicLit: &ast.BasicLit{Kind: token.Token_STRING, Value: "\"hello\""}},
						},
						{
							Name: &ast.Ident{Name: "list"},
							Type: &ast.InputValue_List{List: &ast.List{Type: &ast.List_Ident{Ident: &ast.Ident{Name: "Int"}}}},
							Default: &ast.InputValue_CompositeLit{CompositeLit: &ast.CompositeLit{Value: &ast.CompositeLit_ListLit{
								ListLit: &ast.ListLit{
									List: &ast.ListLit_BasicList{BasicList: &ast.ListLit_Basic{Values: []*ast.BasicLit{
										{Value: "1"},
										{Value: "2"},
										{Value: "3"},
									}}},
								},
							}}},
						},
					},
				},
			},
		}}

		g.generateDirective("Test", false, nil, ts)

		ex := []byte(`NewDirective(graphql.DirectiveConfig{
	Name: "Test",
	Locations: []string{
		"QUERY",
		"FIELD",
		"SCHEMA",
	},
	Args: graphql.FieldConfigArgument{
		"one": &graphql.ArgumentConfig{
			Type: graphql.Int,
			DefaultValue: 1,
		},
		"str": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
			DefaultValue: "hello",
		},
		"list": &graphql.ArgumentConfig{
			Type: graphql.NewList(graphql.Int),
			DefaultValue: []interface{}{1, 2, 3},
		},
	},
})
`)

		compareBytes(subT, ex, g.Bytes())
	})
}

type testCtx struct {
	io.Writer
}

func (ctx testCtx) Open(filename string) (io.WriteCloser, error) { return ctx, nil }

func (ctx testCtx) Close() error { return nil }

func TestGenerator_Generate(t *testing.T) {
	g := &Generator{}

	var b bytes.Buffer
	ctx := compiler.WithContext(context.Background(), testCtx{Writer: &b})
	err := g.Generate(ctx, testDoc, `{"descriptions": true}`)
	if err != nil {
		t.Error(err)
		return
	}

	ex, err := ioutil.ReadAll(exDoc)
	if err != nil {
		t.Error(err)
		return
	}

	compareBytes(t, ex, b.Bytes())
}

func BenchmarkGenerator_Generate(b *testing.B) {
	g := &Generator{}

	var buf bytes.Buffer
	ctx := compiler.WithContext(context.Background(), testCtx{Writer: &buf})

	for i := 0; i < b.N; i++ {
		buf.Reset()

		err := g.Generate(ctx, testDoc, "")
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func ExampleGenerator_Generate() {
	g := new(Generator)

	gqlSrc := `schema {
	query: Query
}

"Query represents the queries this example provides."
type Query {
	hello: String
}`

	doc, err := parser.ParseDoc(token.NewDocSet(), "example", strings.NewReader(gqlSrc), 0)
	if err != nil {
		log.Fatal(err)
		return
	}

	var b bytes.Buffer
	ctx := compiler.WithContext(context.Background(), &testCtx{Writer: &b}) // Pass in an actual
	err = g.Generate(ctx, doc, `{"descriptions": true}`)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(b.String())

	// Output:
	// package main
	//
	// import "github.com/graphql-go/graphql"
	//
	// var Schema graphql.Schema
	//
	// var QueryType = graphql.NewObject(graphql.ObjectConfig{
	// 	Name: "Query",
	//	Fields: graphql.Fields{
	//		"hello": &graphql.Field{
	//			Type: graphql.String,
	//			Resolve: func(p graphql.ResolveParams) (interface{}, error) { return nil, nil }, // TODO
	//		},
	//	},
	//	Description: "Query represents the queries this example provides.",
	// })
}
