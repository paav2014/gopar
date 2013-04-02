// Types
//
// Supports operations on types, such as figuring out the type of a struct
// field or array access. Also figures out the result of a binary expression
// between two types, or a dereference (*) or address-of (&) operation.
//
// Type definitions are fully-defined (Type.final = true), all other types are
// references to them.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
)

var BuiltinTypes map[string]Type

func init() {
	BuiltinTypes = make(map[string]Type, 0)

	builtin := []string{
		"uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64",
		"float32", "float64", "complex64", "complex128", "uint", "int", "uintptr",
		"rune", "byte", "string", "bool", // aliases
	}
	for _, ident := range builtin {
		BuiltinTypes[ident] = TypeDecl(&ast.Ident{Name: ident})
	}

	BuiltinTypes["true"] = BuiltinTypes["bool"]
	BuiltinTypes["false"] = BuiltinTypes["bool"]
	BuiltinTypes["iota"] = BuiltinTypes["int"]

	// builtin functions
	BuiltinTypes["len"] = TypeDecl(&ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{&ast.Field{Type: &ast.Ident{Name: "int"}}}}})
	// make is handled dynamically in types.go:TypeOf
}

type Type interface {
	Complete(Resolver)
	Definition() ast.Node
	Fields() []string  // return an ordered list of all fields in this type
	Field(string) Type // return .field's type
	Dereference() Type // the return type of a *dereference operation
	IndexKey() Type    // the return type of an [index] or <-chan operation
	IndexValue() Type
	Call() []Type                // the return types of calling this type
	Math(Type, token.Token) Type // outcome of any math operation with another type
	String() string              // a representation of this type
	CType() string               // type signature (int* for slice)
	CDecl() string               // type definition (typedef struct {} something)
}

type BaseType struct {
	ast.Node // definition node
}

func newBaseType(node ast.Node) *BaseType {
	return &BaseType{Node: node}
}

func (t *BaseType) Complete(resolver Resolver) {
	return
}

func (t *BaseType) Definition() ast.Node {
	return t.Node
}

func (t *BaseType) Fields() []string {
	return nil
}

func (t *BaseType) Field(name string) Type {
	return nil
}

func (t *BaseType) Dereference() Type {
	return nil
}

func (t *BaseType) IndexKey() Type {
	return nil
}

func (t *BaseType) IndexValue() Type {
	return nil
}

func (t *BaseType) Call() []Type {
	return nil
}

// The declaration of the type, such as "typedef Pixel struct {}"
func (t *BaseType) CDecl() string {
	return "UNKNOWN DECL"
}

// The name of the type, such as "*Pixel" or "int32[]"
func (t *BaseType) CType() string {
	return "UNKNOWN TYPE"
}

func (t *BaseType) Math(other Type, op token.Token) Type {
	return BinaryOp(t, op, other)
}

func (typ *BaseType) String() string {
	var buffer bytes.Buffer
	switch t := typ.Node.(type) {
	case *ast.Ident:
		buffer.WriteString(t.Name)
	case *ast.BasicLit:
		buffer.WriteString(t.Value)
	default:
		buffer.WriteString(fmt.Sprintf("Type<%T %+v>", t, t))
	}
	return buffer.String()
}

type ConstType struct {
	*BaseType
	typ Type // actual type of this constant
}

func newConstType(node ast.Node) *ConstType {
	return &ConstType{
		BaseType: newBaseType(node),
	}
}

func (t *ConstType) Complete(resolver Resolver) {
	switch t.Node.(*ast.BasicLit).Kind {
	case token.FLOAT:
		t.typ = resolver("float64")
	case token.INT:
		t.typ = resolver("int")
	case token.STRING:
		t.typ = resolver("string")
	}
	return
}

func (t *ConstType) Math(other Type, op token.Token) Type {
	return BinaryOp(t.typ, op, other)
}

func (t *ConstType) String() string {
	return fmt.Sprintf("%s=%s", t.typ.String(), t.Node.(*ast.BasicLit).Value)
}

type StructType struct {
	*BaseType
	fieldOrder []string
	fields     map[string]Type
}

func newStructType(node ast.Node) *StructType {
	return &StructType{
		BaseType:   newBaseType(node),
		fieldOrder: make([]string, 0),
		fields:     make(map[string]Type),
	}
}

// fill in all struct fields
func (t *StructType) Complete(resolver Resolver) {
	switch e := t.Node.(type) {
	case *ast.StructType:
		for _, field := range e.Fields.List {
			fieldTyp := TypeOf(field.Type, resolver)
			// Embedded fields
			// *Struct1
			// Struct2
			// abc.Struct3
			if len(field.Names) == 0 {
				var ig IdentifierGroup
				AccessIdentBuild(&ig, field.Type, nil)
				name := ig.group[len(ig.group)-1].id
				t.addField(name, fieldTyp)
			} else {
				for _, name := range field.Names {
					t.addField(name.Name, fieldTyp)
				}
			}
		}
	}
}

func (t *StructType) addField(name string, typ Type) {
	t.fields[name] = typ
	t.fieldOrder = append(t.fieldOrder, name)
	return
}

func (t *StructType) Fields() []string {
	return t.fieldOrder
}

func (t *StructType) Field(name string) Type {
	return t.fields[name]
}

func (t *StructType) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct {")
	for i, field := range t.Fields() {
		buffer.WriteString(fmt.Sprintf("%s=%s", field, t.Field(field).String()))
		if i != len(t.Fields())-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}

// an array, list, map or chan type
type IndexedType struct {
	*BaseType
	key   Type
	value Type
}

func newIndexedType(node ast.Node) *IndexedType {
	return &IndexedType{
		BaseType: newBaseType(node),
	}
}

// fill in key and value sections
func (typ *IndexedType) Complete(resolver Resolver) {
	switch t := typ.Node.(type) {
	case *ast.ArrayType:
		typ.key = TypeOf(&ast.Ident{Name: "int"}, resolver)
		typ.value = TypeOf(t.Elt, resolver)
	case *ast.MapType:
		typ.key = TypeOf(t.Key, resolver)
		typ.value = TypeOf(t.Value, resolver)
	case *ast.ChanType:
		typ.value = TypeOf(t.Value, resolver)
	}
	return
}

func (t *IndexedType) IndexKey() Type {
	return t.key
}

func (t *IndexedType) IndexValue() Type {
	return t.value
}

func (typ *IndexedType) String() string {
	var buffer bytes.Buffer
	switch t := typ.Node.(type) {
	case *ast.ArrayType:
		buffer.WriteString("[")
		if t.Len != nil {
			buffer.WriteString(typ.IndexKey().String())
		}
		buffer.WriteString("]")
	case *ast.MapType:
		buffer.WriteString("map[")
		buffer.WriteString(typ.IndexKey().String())
		buffer.WriteString("]")
	case *ast.ChanType:
		if t.Dir == ast.SEND {
			buffer.WriteString("->")
		} else if t.Dir == ast.RECV {
			buffer.WriteString("<-")
		}
		buffer.WriteString("chan ")
	}
	buffer.WriteString(typ.value.String())
	return buffer.String()
}

// a pointer type
type PointerType struct {
	*BaseType
	inner Type
}

func newPointerType(node ast.Node) *PointerType {
	return &PointerType{
		BaseType: newBaseType(node),
	}
}

// Resolve the inner type
func (t *PointerType) Complete(resolver Resolver) {
	expr := t.Node.(*ast.StarExpr).X
	t.inner = TypeOf(expr, resolver)
	return
}

func (t *PointerType) Dereference() Type {
	return t.inner
}

func (t *PointerType) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("*")
	buffer.WriteString(t.inner.String())
	return buffer.String()
}

// a function
type FuncType struct {
	*BaseType
	params  []Type
	results []Type
}

func newFuncType(node ast.Node) *FuncType {
	return &FuncType{
		BaseType: newBaseType(node),
		params:   nil,
		results:  nil,
	}
}

// fill in params and results
func (t *FuncType) Complete(resolver Resolver) {
	expr := t.Node.(*ast.FuncType)
	if expr.Params != nil {
		if t.params != nil {
			panic("Already Completed")
		}
		for _, arg := range expr.Params.List {
			argType := TypeOf(arg.Type, resolver)
			for _ = range arg.Names {
				t.params = append(t.params, argType)
			}
		}
	}
	if expr.Results != nil {
		if t.results != nil {
			panic("Already Completed")
		}
		for _, result := range expr.Results.List {
			resultType := TypeOf(result.Type, resolver)
			i := len(result.Names)
			if i == 0 {
				i = 1
			}
			for j := 0; j < i; j++ {
				t.results = append(t.results, resultType)
			}
		}
	}
	return
}

func (t *FuncType) Call() []Type {
	return t.results
}

func (t *FuncType) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("func (")
	for i, param := range t.params {
		buffer.WriteString(param.String())
		if i < len(t.params)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(")")
	if len(t.results) > 0 {
		buffer.WriteString(" (")
		for i, result := range t.results {
			buffer.WriteString(result.String())
			if i < len(t.results)-1 {
				buffer.WriteString(", ")
			}
		}
		buffer.WriteString(")")
	}
	return buffer.String()
}

type PackageType struct {
	*BaseType
	members map[string]Type
}

func newPackageType(node ast.Node) Type {
	return &PackageType{
		BaseType: newBaseType(node),
		members:  make(map[string]Type),
	}
}

func (t *PackageType) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("package ")
	buffer.WriteString(t.Node.(*ast.ImportSpec).Path.Value)
	return buffer.String()
}

// The outcome of a binary operation
func BinaryOp(X Type, op token.Token, Y Type) Type {
	switch op {
	case token.LAND, token.LOR, token.NEQ, token.LEQ, token.GEQ, token.EQL,
		token.LSS, token.GTR:
		return BuiltinTypes["bool"]
	default:
		// Binary operations are always between two of the same types, unless
		// shifting. Untyped constants are converted to the type of the other
		// operand
		_, xConst := X.(*ConstType)
		_, yConst := Y.(*ConstType)
		switch {
		case !xConst && !yConst:
			return X
		case xConst && !yConst:
			return X
		case yConst && !xConst:
			return Y
		default:
			// TODO: see http://golang.org/ref/spec#Constant_expressions
			return X
		}
	}
	return nil
}

// Create a new type from a declaration Node
func TypeDecl(expr ast.Node) Type {
	switch n := expr.(type) {
	case *ast.ChanType, *ast.ArrayType, *ast.MapType:
		return newIndexedType(n)
	case *ast.StructType, *ast.InterfaceType:
		return newStructType(n)
	case *ast.StarExpr:
		return newPointerType(n)
	case *ast.FuncType:
		return newFuncType(n)
	case *ast.Ident:
		return newBaseType(n)
	case *ast.ImportSpec:
		return newPackageType(n)
	}
	return nil
}

// Takes an identifier, returns the node that defines it. This should search all
// scopes up to the package level.
type Resolver func(ident string) Type

func TypeOf(expr ast.Node, resolver Resolver) Type {
	switch t := expr.(type) {
	case *ast.CallExpr:
		var fnType Type
		switch f := t.Fun.(type) {
		case *ast.Ident:
			// hook special functions
			if f.Name == "make" {
				fnType = newFuncType(&ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{&ast.Field{Type: t.Args[0]}}}})
				fnType.Complete(resolver)
			} else if builtin, ok := BuiltinTypes[f.Name]; ok {
				// type conversion
				// int32(x)
				return builtin
			} else {
				fnType = resolver(f.Name)
			}
		case *ast.FuncLit:
			fnType = newFuncType(f.Type)
			fnType.Complete(resolver)
		}
		return fnType
	case *ast.Ident:
		return resolver(t.Name)
	case *ast.BasicLit:
		constTyp := newConstType(t)
		constTyp.Complete(resolver)
		return constTyp
	case *ast.IndexExpr:
		indexer := TypeOf(t.X, resolver)
		indexer.Complete(resolver)
		return indexer.IndexValue()
	case *ast.UnaryExpr:
		// &something
		switch t.Op {
		case token.AND:
			refTyp := newPointerType(&ast.StarExpr{X: t.X})
			refTyp.Complete(resolver)
			return refTyp
		// <-chan
		case token.ARROW:
			chanTyp := TypeOf(t.X, resolver)
			return chanTyp.IndexValue()
		case token.NOT:
			return BuiltinTypes["bool"]
		}
		return TypeOf(t, resolver)
	case *ast.StarExpr:
		ptrType := TypeOf(t.X, resolver)
		return ptrType.Dereference()
	case *ast.CompositeLit:
		// Something{}
		return TypeOf(t.Type, resolver)
	case *ast.BinaryExpr:
		xTyp := TypeOf(t.X, resolver)
		yTyp := TypeOf(t.Y, resolver)
		result := xTyp.Math(yTyp, t.Op)
		fmt.Println(result, "=", xTyp.String(), t.Op, yTyp.String())
		return result
	case *ast.ArrayType, *ast.ChanType, *ast.MapType:
		indexTyp := newIndexedType(t)
		indexTyp.Complete(resolver)
		return indexTyp
	case *ast.SelectorExpr:
		innerTyp := TypeOf(t.X, resolver)
		return innerTyp.Field(t.Sel.Name)
	default:
		fmt.Printf("Unhandled TypeOf(%T %+v)\n", expr, expr)
	}
	return nil
}

// Helper functions for constructing C/OpenCL structures:
// http://golang.org/ref/spec#Size_and_alignment_guarantees
// http://www.khronos.org/registry/cl/sdk/1.1/docs/man/xhtml/attributes-types.html
// https://code.google.com/p/go/source/browse/go/types/sizes.go?repo=exp
func SizeOf(typ Type) int64 {
	return 1
}

func AlignOf(typ Type) int64 {
	return 1
}
