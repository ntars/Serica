package main
import ("fmt"; "go/parser"; "go/token"; "go/ast"; "os")
func main(){ fset:=token.NewFileSet(); f,err:=parser.ParseFile(fset,"bootstrap/semantic.go",nil,0); if err!=nil{panic(err)}; for _,d:=range f.Decls { if fd,ok:=d.(*ast.FuncDecl); ok && fd.Name.Name=="collectRuntimeTextReturnParameters" { ast.Fprint(os.Stdout,fset,fd.Body,ast.NotNilFilter); fmt.Println(); return } } }
