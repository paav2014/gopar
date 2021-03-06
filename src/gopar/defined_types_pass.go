// Defined types pass
//
// Gather all user-defined types for a package

package main

import (
	"fmt"
	"go/ast"
	"strings"
)

type DefinedTypesData struct {
	defined  map[string]Type
	embedded []Type // used when other packages import . "something"
}

func NewDefinedTypesData() *DefinedTypesData {
	d := &DefinedTypesData{
		defined: make(map[string]Type),
	}
	return d
}

// storage for future declarations
type futureDecl struct {
	names      []string
	exprs      []ast.Node
	isDecl     bool // should we use TypeOfDecl or TypeOf?
	makeShadow bool // should the declaration shadow the TypeOfDecl(exprs)?
}

type DefinedTypesPass struct {
	BasePass
}

func NewDefinedTypesPass() *DefinedTypesPass {
	return &DefinedTypesPass{
		BasePass: NewBasePass(),
	}
}

func (pass *DefinedTypesPass) GetPassType() PassType {
	return DefinedTypesPassType
}

func (pass *DefinedTypesPass) GetPassMode() PassMode {
	return ModulePassMode
}

func (pass *DefinedTypesPass) GetDependencies() []PassType {
	return []PassType{}
}

func (pass *DefinedTypesPass) RunModulePass(file *ast.File, p *Package) (modified bool, err error) {
	data := NewDefinedTypesData()
	pass.SetResult(p, data)

	var methods []*ast.FuncDecl
	var future = make(map[string]*futureDecl)
	// make everything a FutureType to be resolved later
	Define := func(name string, t Type) {
		if val, exists := data.defined[name]; exists {
			// if they're packages with the same name, ignore it
			pkg1, ok1 := val.(*PackageType)
			pkg2, ok2 := val.(*PackageType)
			if !ok1 || !ok2 || pkg1.path != pkg2.path {
				err = fmt.Errorf("Redefining identifier %s = %s with %s", name, val.String(), t.String())
				return
			}
		}
		delete(future, name)
		data.defined[name] = t
	}

	Future := func(fd *futureDecl) {
		for _, name := range fd.names {
			if name == "" {
				panic(fmt.Sprintf("Empty name in %+v", fd))
			}
			future[name] = fd
		}
	}

	//resolver := MakeResolver(nil, p, pass.compiler)
	// Make a custom resolver that will recursively fill in types. There are two
	// recursive definition problem spots:
	// type A struct {
	//   next *A
	// }
	// type F func(F)
	// Solve these by adding an additional TypeOf parameter to not complete the
	// newly created type automatically. Only complete the type after it has been
	// added to the definitions table.
	var resolver Resolver
	resolver = func(name string) Type {
		// check the current package
		// if match doesn't have a type yet, resolve it and save it
		if name == "" {
			panic(1)
		}
		if futureDecl, ok := future[name]; ok {
			if len(futureDecl.exprs) < len(futureDecl.names) {
				// multi-assign
				if len(futureDecl.exprs) != 1 {
					panic(fmt.Sprintf("invalid multi-assign: %d to %d", len(futureDecl.exprs), len(futureDecl.names)))
				}
				result := TypeOf(futureDecl.exprs[0], resolver).(*MultiType).Expand()
				for i, name := range futureDecl.names {
					Define(name, result[i])
				}
			} else {
				// normal assign
				for i, name := range futureDecl.names {
					var result Type
					if futureDecl.isDecl {
						result = TypeDecl(futureDecl.exprs[i], resolver)
					} else {
						result = TypeOf(futureDecl.exprs[i], resolver)
					}
					var shadowType Type
					if futureDecl.makeShadow {
						shadowType = newShadowType(result)
					}
					if shadowType != nil {
						Define(name, shadowType)
					} else {
						Define(name, result)
					}
					// only complete after we've defined it so recursive definitions can
					// be picked up
					if futureDecl.isDecl {
						result.Complete(resolver)
						if shadowType != nil {
							shadowType.Complete(resolver)
						}
					}
				}
			}
		}

		// return the type
		if typ, ok := data.defined[name]; ok {
			return typ
		}

		// check embedded "." packages
		for _, embedded := range data.embedded {
			if identTyp := embedded.Field(name); identTyp != nil {
				return identTyp
			}
		}

		// check builtin types
		if identTyp, ok := BuiltinTypes[name]; ok {
			return identTyp
		}
		return nil
	}

	// Top-level definitions don't have to be done in any order
	for _, decl := range file.Decls {
		switch t := decl.(type) {
		case *ast.FuncDecl:
			if t.Recv != nil {
				methods = append(methods, t)
			} else {
				Future(&futureDecl{
					names:  []string{t.Name.Name},
					exprs:  []ast.Node{t},
					isDecl: true,
				})
			}
		case *ast.GenDecl:
			var prevType ast.Expr // used for constants
			var prevValues []ast.Expr
			for _, spec := range t.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					// new types should shadow(TypeOfDecl(s.Type)), not be a direct link
					// to them
					Future(&futureDecl{
						names:      []string{s.Name.Name},
						exprs:      []ast.Node{s.Type},
						isDecl:     true,
						makeShadow: true,
					})
				case *ast.ValueSpec:
					// a single line of a declaration block
					// const (
					//   a, b int = 1, 2 (s.Type = int, s.Names = [a,b], s.Values = [1,2])
					//   c, d = iota, iota*2 (s.Type = nil, s.Names = [c,d], s.Values = [iota,iota*2])
					//   e, f (s.Type = nil, s.Names = [e,f], s.Values = [])
					// )
					// var a, b = pkgB.Fun()
					// every ValueSpec set the prevValues and prevType values if they exist
					if s.Type != nil || len(s.Values) > 0 {
						prevType = s.Type
						// if new values are defined they may be untyped constants (type = nil)
						prevValues = s.Values
					}
					var names []string
					var exprs []ast.Node
					for _, name := range s.Names {
						if prevType != nil {
							exprs = append(exprs, prevType)
						}
						names = append(names, name.Name)
					}
					if prevType == nil {
						for _, e := range prevValues {
							exprs = append(exprs, e)
						}
					}
					Future(&futureDecl{
						names:  names,
						exprs:  exprs,
						isDecl: prevType != nil,
					})
				case *ast.ImportSpec:
					// TODO: package imports should only be for this file
					// http://golang.org/ref/spec#Declarations_and_scope
					var name string
					var path string
					if s.Name != nil {
						name = s.Name.Name
						path = name
					} else {
						// pull from import path
						path = s.Path.Value
						idx := strings.LastIndex(path, "/")
						if idx == -1 {
							name = path
						} else {
							name = path[idx:]
						}
					}
					name = name[1 : len(name)-1] // strip off ["] in the back and [/"] in front
					path = path[1 : len(path)-1]
					if name != "_" {
						// attach the types found in the other package here
						otherPackage := pass.compiler.project.get(path)
						otherPackageTypes, ok := pass.compiler.GetPassResult(DefinedTypesPassType, otherPackage).(*DefinedTypesData)

						var packageResolver Resolver
						if ok {
							packageResolver = func(name string) Type {
								pkgTyp := otherPackageTypes.defined[name]
								if *verbose {
									fmt.Printf("Cross-package lookup: %s.%s = %s\n", path, name, pkgTyp)
								}
								return pkgTyp
							}
						} else {
							// package not found
							err = fmt.Errorf("Package not found: %s", path)
							return
						}
						pkgType := newPackageType(s)
						pkgType.Complete(packageResolver)
						if name == "." {
							data.embedded = append(data.embedded, pkgType)
						} else {
							// define packages right away
							Define(name, pkgType)
						}
					}
				}
			}
		default:
			if *verbose {
				fmt.Printf("Unhandled Decl %T %+v", decl, decl)
			}
		}
	}

	var undefined []string
	for name := range future {
		undefined = append(undefined, name)
	}

	for _, name := range undefined {
		typ := resolver(name)
		if typ == nil {
			err = fmt.Errorf("Could not resolve defined type %s", name)
			return
		}
		if *verbose {
			fmt.Printf("%s = %s\n", name, typ)
		}
	}

	// fill in methods
	for _, method := range methods {
		recvTyp := TypeOfDecl(method.Recv.List[0].Type, resolver)

		methodTyp := newMethodType(method.Type, method, recvTyp)
		methodTyp.Complete(resolver)
		if *verbose {
			fmt.Printf("Adding method %+v to %+v\n", method, methodTyp)
		}
		recvTyp.AddMethod(method.Name.Name, methodTyp)
	}
	if *verbose {
		for name, typ := range data.defined {
			fmt.Printf("%s = %v\n", name, typ)
		}
	}
	return
}
