// Package omarchy provides read-only adapters for optional desktop integration.
package omarchy

import(
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/pelletier/go-toml/v2"
	"github.com/OWNER/lenger/internal/ui/theme"
	"github.com/OWNER/lenger/internal/ui/tokens"
)

// ThemeSource isolates desktop palette lookup from the UI.
type ThemeSource interface{Load(context.Context)(tokens.Palette,error);Watch(context.Context,func(tokens.Palette)error)error}

// Source reads the active theme directory, following its current symlink.
type Source struct{Directory string}

// DefaultSource honors XDG_CONFIG_HOME through os.UserConfigDir.
func DefaultSource()Source{dir,_:=os.UserConfigDir();return Source{Directory:filepath.Join(dir,"omarchy","current","theme")}}

// Load reads colors.toml, Alacritty, then Ghostty with a safe bundled fallback.
func(s Source)Load(ctx context.Context)(tokens.Palette,error){
	p,_:=theme.Bundled("midnight");if err:=ctx.Err();err!=nil{return p,err}
	for _,file:=range []string{"colors.toml","alacritty.toml","ghostty.conf"}{
		data,err:=os.ReadFile(filepath.Join(s.Directory,file));if err!=nil{continue}
		values:=make(map[string]string)
		if file=="ghostty.conf"{scanner:=bufio.NewScanner(strings.NewReader(string(data)));for scanner.Scan(){k,v,ok:=strings.Cut(scanner.Text(),"=");if ok{values[strings.TrimSpace(k)]=strings.Trim(strings.TrimSpace(v),"\"'")}}}else{
			var raw map[string]any;if toml.Unmarshal(data,&raw)!=nil{continue};flatten(raw,values)
		}
		apply:=func(dst *string,keys ...string){for _,key:=range keys{if v:=values[key];v!=""{if !strings.HasPrefix(v,"#"){v="#"+v};if _,err:=theme.RGB(v);err==nil{*dst=v;return}}}}
		apply(&p.BG,"background","colors.primary.background");apply(&p.Text,"foreground","colors.primary.foreground");apply(&p.Accent,"accent","cyan","colors.normal.cyan");apply(&p.Accent2,"magenta","colors.normal.magenta");apply(&p.Success,"green","colors.normal.green");apply(&p.Danger,"red","colors.normal.red");apply(&p.Warning,"yellow","colors.normal.yellow")
		p.Surface=p.BG;p.Raised=p.BG;p.BorderFocus=p.Accent;p.Series=[6]string{p.Accent,p.Accent2,p.Info,p.Success,p.Warning,p.Danger};p.Name="omarchy";return theme.Readable(p),nil
	}
	return p,nil
}

func flatten(raw map[string]any,out map[string]string){for key,value:=range raw{switch v:=value.(type){case string:out[key]=v;case map[string]any:child:=make(map[string]string);flatten(v,child);for k,s:=range child{out[key+"."+k]=s}}}}

// Watch checks desktop files only, never scheduling UI animation ticks.
func(s Source)Watch(ctx context.Context,emit func(tokens.Palette)error)error{
	previous,err:=s.Load(ctx);if err!=nil{return err};timer:=time.NewTicker(time.Second);defer timer.Stop()
	for{select{case <-ctx.Done():return ctx.Err();case <-timer.C:p,err:=s.Load(ctx);if err==nil&&p!=previous{previous=p;if err:=emit(p);err!=nil{return err}}}}
}

// Setup prints integration suggestions without changing desktop configuration.
func Setup()string{return fmt.Sprintln("# Review before adding to ~/.config/hypr/bindings.conf:\n# Suggested unused binding: Super+Alt+D\n# Launch lenger with your installed Omarchy TUI helper; helper compatibility is not verified.\n# No files were changed.")}
