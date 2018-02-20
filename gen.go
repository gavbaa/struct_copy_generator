// The doc command prints the doc comment of a package-level object.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/types"
	"golang.org/x/tools/go/loader"
	"log"
	"strings"
	"os"
	"errors"
)

func structIt(pkgPath string, name string) *types.Named {
	// The loader loads a complete Go program from source code.
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Import(pkgPath)
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err) // load error
	}

	// Find the package and package-level object.
	pkg := lprog.Package(pkgPath).Pkg
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		log.Fatalf("%s.%s not found", pkg.Path(), name)
	}

	return obj.Type().(*types.Named)
}

func writeCopyFunc(t1 *types.Named, t2 *types.Named, outputPackageName string) (string, error) {
	s1 := t1.Underlying().(*types.Struct)
	s2 := t2.Underlying().(*types.Struct)

	// spew.Dump(t1.Obj().Pkg().Name())

	imports := map[string]string{}
	imports[t1.Obj().Pkg().Path()] = "one"
	imports[t2.Obj().Pkg().Path()] = "two"

	funcHeader := fmt.Sprintf("func Copy%s%sTo%s%s(a *one.%s, b *two.%s) {\n",
		t1.Obj().Pkg().Name(),
		t1.Obj().Name(),
		t2.Obj().Pkg().Name(),
		t2.Obj().Name(),
		t1.Obj().Name(),
		t2.Obj().Name(),
	)

	i := 0
	tmpCounter := 1
	funcTop := ""
	fieldBody := ""
	for i < s1.NumFields() {
		f1 := s1.Field(i)
		j := 0
		for j < s2.NumFields() {
			f2 := s2.Field(j)
			if strings.ToLower(f1.Name()) == strings.ToLower(f2.Name()) {
				// Fields match

				// Do soft type conversions.
				// println("f1.Type", f1.Type().String(), f2.Type().String())
				f1Type := f1.Type().String()
				f2Type := f2.Type().String()
				if f1Type == f2Type {
					fieldBody += fmt.Sprintf("b.%s = a.%s\n", f2.Name(), f1.Name())
				} else if f1Type == "int" && f2Type == "int64" {
					fieldBody += fmt.Sprintf("b.%s = int64(a.%s)\n", f2.Name(), f1.Name())
				} else if f1Type == "int64" && f2Type == "int" {
					fieldBody += fmt.Sprintf("b.%s = int(a.%s)\n", f2.Name(), f1.Name())
				} else if f1Type == "int" && f2Type == "string" {
					imports["strconv"] = ""
					fieldBody += fmt.Sprintf("b.%s = strconv.FormatInt(a.%s, 10)\n", f2.Name(), f1.Name())
				} else if f1Type == "int64" && f2Type == "string" {
					imports["strconv"] = ""
					fieldBody += fmt.Sprintf("b.%s = strconv.FormatInt(a.%s, 10)\n", f2.Name(), f1.Name())
				} else if f1Type == "time.Time" && f2Type == "*github.com/golang/protobuf/ptypes/timestamp.Timestamp" {
					// TODO Error handling
					imports["github.com/golang/protobuf/ptypes"] = ""
					funcTop += fmt.Sprintf("tmp%d, _ := ptypes.TimestampProto(a.%s)\n", tmpCounter, f1.Name())
					fieldBody += fmt.Sprintf("b.%s = tmp%d\n", f1.Name(), tmpCounter)
					tmpCounter += 1
				} else {
					println(fmt.Sprintf("Couldn't find a reasonable type match: %s, %s = %s", f1.Name(), f1.Type().String(), f2.Type().String()))
				}
			}
			j += 1
		}
		i += 1
	}

	importStr := ""
	for importPath, importAlias := range imports {
		importStr += fmt.Sprintf("import %s \"%s\"\n", importAlias, importPath)
	}

	out := fmt.Sprintf("package %s\n", outputPackageName)
	out += importStr
	out += funcHeader
	out += funcTop
	out += fieldBody
	out += "}\n"

	cleanOutput, err := gofmt(out)
	if err != nil {
		return "", errors.New("gofmt failed: " + err.Error() + ".  Pre-formatted output is as follows:\n\n" + out)
	}
	return cleanOutput, nil
}

func main() {
	outPkg := flag.String("out-pkg-name", "x", "Customize the package name for this file")
	flag.Parse()
	args := flag.Args()
	if len(args) < 4 {
		log.Fatal("Usage: doc <package1> <object1> <package2> <object2>")
	}

	pkgPath1, name1 := args[0], args[1]
	t1 := structIt(pkgPath1, name1)
	//spew.Dump(t1)
	//spew.Dump(t1.Underlying().(*types.Struct).Field(1).Type())
	//spew.Dump(t1.Underlying().(*types.Struct).Field(17).Type())

	pkgPath2, name2 := args[2], args[3]
	t2 := structIt(pkgPath2, name2)

	funcOut, err := writeCopyFunc(t1, t2, *outPkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
	os.Stdout.WriteString(funcOut)
}
