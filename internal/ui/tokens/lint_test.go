package tokens

import(
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestTokenOnlyStyling(t *testing.T){root:=filepath.Join("..","..","..");hex:=regexp.MustCompile(`#[0-9a-fA-F]{6}\b`);err:=filepath.WalkDir(root,func(path string,d fs.DirEntry,err error)error{if err!=nil{return err};if d.IsDir(){return nil};if !strings.HasSuffix(path,".go")||strings.HasSuffix(path,"_test.go"){return nil};normalized:=filepath.ToSlash(path);if strings.Contains(normalized,"/ui/tokens/")||strings.Contains(normalized,"/ui/theme/"){return nil};fset:=token.NewFileSet();file,err:=parser.ParseFile(fset,path,nil,0);if err!=nil{return err};ast.Inspect(file,func(n ast.Node)bool{switch node:=n.(type){case *ast.BasicLit:if node.Kind==token.STRING{value,_:=strconv.Unquote(node.Value);if hex.MatchString(value){t.Errorf("%s: color outside tokens/theme",fset.Position(node.Pos()))}};case *ast.CallExpr:if sel,ok:=node.Fun.(*ast.SelectorExpr);ok&&(strings.HasPrefix(sel.Sel.Name,"Padding")||strings.HasPrefix(sel.Sel.Name,"Margin")){for _,arg:=range node.Args{if literal,ok:=arg.(*ast.BasicLit);ok&&literal.Value!="0"{t.Errorf("%s: non-token spacing",fset.Position(arg.Pos()))}}}};return true});return nil});if err!=nil{t.Fatal(err)}}
