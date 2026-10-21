package gallery

import(
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"charm.land/lipgloss/v2"
	"github.com/OWNER/lenger/internal/ui/theme"
)

var update=flag.Bool("update",false,"explicitly update reviewed gallery goldens")

func TestLayout(t *testing.T){t.Setenv("NO_COLOR","1");for _,name:=range theme.Names(){for _,size:=range Sizes(){t.Run(fmt.Sprintf("%s/%dx%d",name,size.Width,size.Height),func(t *testing.T){frames,err:=Frames(context.Background(),name,size);if err!=nil{t.Fatal(err)};for _,frame:=range frames{lines:=strings.Split(frame.Content,"\n");if len(lines)!=size.Height{t.Errorf("%s: %d lines, want %d",frame.Name,len(lines),size.Height)};for n,line:=range lines{if lipgloss.Width(line)>size.Width{t.Errorf("%s line %d overflows",frame.Name,n)}}}})}}}

func TestGolden(t *testing.T){t.Setenv("NO_COLOR","1");for _,name:=range theme.Names(){for _,size:=range Sizes(){frames,err:=Frames(context.Background(),name,size);if err!=nil{t.Fatal(err)};for _,frame:=range frames{path:=filepath.Join("..","..","testdata","golden",fmt.Sprintf("%s-%dx%d-%s.txt",name,size.Width,size.Height,frame.Name));if *update{if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{t.Fatal(err)};if err:=os.WriteFile(path,[]byte(frame.Content),0644);err!=nil{t.Fatal(err)}};want,err:=os.ReadFile(path);if err!=nil{t.Fatalf("golden missing: %s; render/review first, then use -update: %v",path,err)};if string(want)!=frame.Content{t.Errorf("golden mismatch: %s",path)}}}}}
