package app

import(
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/OWNER/lenger/internal/config"
	"github.com/OWNER/lenger/internal/core/services"
	"github.com/OWNER/lenger/internal/engine/fake"
	"github.com/OWNER/lenger/internal/ui/theme"
)

func fixtureModel(t *testing.T)*Model{t.Helper();ctx,cancel:=context.WithCancel(context.Background());t.Cleanup(cancel);s:=services.New(fake.New());p,err:=theme.Bundled("midnight");if err!=nil{t.Fatal(err)};c:=config.Defaults();c.UI.ReduceMotion=true;c.UI.Theme="midnight";m:=New(ctx,s,c,p);batch,err:=s.Snapshot(ctx);if err!=nil{t.Fatal(err)};m.Gallery(batch,1,"",nil);t.Cleanup(m.cancel);return m}

func TestIdleUpdateSchedulesNoTick(t *testing.T){m:=fixtureModel(t);_,cmd:=m.Update(struct{}{});if cmd!=nil{t.Fatal("idle update scheduled a command")}}
func TestReducedTabSwitchNoTick(t *testing.T){m:=fixtureModel(t);_,cmd:=m.Update(tea.KeyPressMsg{Code:'3',Text:"3"});if cmd!=nil||m.tab!=2{t.Fatal("reduced navigation scheduled animation or failed")}}
func TestSelectionAndFilter(t *testing.T){m:=fixtureModel(t);_,_=m.Update(tea.KeyPressMsg{Code:' ',Text:" "});if !m.selected[m.items()[0].ID]{t.Fatal("space did not select")};m.filter="redis";m.clamp();if len(m.items())!=1||m.items()[0].Name()!="redis:7.4"{t.Fatal("filter failed")}}
func TestPermissionErrorDoesNotDisplayFixtureResources(t *testing.T){m:=fixtureModel(t);_,_=m.Update(snapshotMsg{err:errPermission{}});text:=m.Render();if !strings.Contains(text,"Docker socket access was denied")||strings.Contains(text,"redis:7.4"){t.Fatal("permission screen did not replace resources")}}
type errPermission struct{}
func(errPermission)Error()string{return "permission denied"}

func TestProgramWithASCIIOutput(t *testing.T){m:=fixtureModel(t);ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second);defer cancel();var output bytes.Buffer;p:=tea.NewProgram(m,tea.WithContext(ctx),tea.WithInput(strings.NewReader("q")),tea.WithOutput(&output),tea.WithWindowSize(80,24),tea.WithColorProfile(colorprofile.Ascii));if _,err:=p.Run();err!=nil{t.Fatal(err)}}
