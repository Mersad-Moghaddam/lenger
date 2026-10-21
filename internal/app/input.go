package app

import(
	"fmt"
	"strconv"
	"strings"
	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"
	"github.com/OWNER/lenger/internal/core/services"
	"github.com/OWNER/lenger/internal/ui/components"
	"github.com/OWNER/lenger/internal/ui/tokens"
)

type action struct{name,key,command string}
func actions()[]action{return []action{{"Dashboard","1",""},{"Images","2","docker image ls"},{"Pull image","p","docker pull <reference>"},{"Inspect image","i","docker image inspect <id>"},{"Remove images","d","docker image rm <id>"},{"Filter images","/","docker image ls"},{"Refresh resources","R","docker image ls; docker ps -a"},{"Job history","J",""},{"Help","?",""}}}
func(m *Model)paletteActions()[]action{
	all:=actions();for _,image:=range m.items(){all=append(all,action{"Image: "+image.Name(),"image:"+image.ID,"docker image inspect "+image.ID})}
	if m.input.Value()==""{return all};names:=make([]string,len(all));for n,a:=range all{names[n]=a.name};matches:=fuzzy.Find(m.input.Value(),names);result:=make([]action,0,len(matches));for _,match:=range matches{result=append(result,all[match.Index])};return result
}
func(m *Model)openInput(kind string)tea.Cmd{m.overlay=kind;m.input.Reset();m.input.Placeholder="";if kind=="pull-input"{m.input.Placeholder="redis:7.4"};if kind=="filter"{m.input.SetValue(m.filter)};m.paletteCursor=0;return m.input.Focus()}
func(m *Model)closeOverlay(){m.overlay="";m.input.Blur()}

func(m *Model)key(msg tea.KeyPressMsg)tea.Cmd{
	k:=msg.String()
	if m.overlay!=""{return m.overlayKey(msg)}
	for action,binding:=range m.config.Keys{if binding==k{switch action{case "pull":k="p";case "inspect":k="i";case "delete":k="d";case "filter":k="/";case "palette":k=":";case "help":k="?";case "quit":k="q"}}}
	if strings.HasPrefix(k,"image:"){id:=strings.TrimPrefix(k,"image:");m.tab=1;for n,i:=range m.items(){if i.ID==id{m.cursor=n;return m.inspect()}};return nil}
	if number,err:=strconv.Atoi(k);err==nil&&number>=1&&number<=7{m.tab=number-1;m.detail=false;return nil}
	switch k{
	case "q":return m.quit()
	case "esc":if m.detail{m.detail=false}else if m.filter!=""{m.filter="";m.clamp()};return nil
	case "?":m.overlay="help"
	case ":","ctrl+p":return m.openInput("palette")
	case "p":return m.openInput("pull-input")
	case "J":m.overlay="jobs"
	case "R":if !m.refreshing{return m.refresh()}
	case "D":m.compact=!m.compact
	case "/":m.tab=1;return m.openInput("filter")
	case "o":if m.tab==1{m.sortMode=(m.sortMode+1)%3;m.cursor=0}
	case "j","down","shift+down":if m.tab==1{items:=m.items();if k=="shift+down"&&len(items)>0{m.selected[items[m.cursor].ID]=true};m.cursor++;m.clamp();if k=="shift+down"&&len(items)>0{m.selected[items[m.cursor].ID]=true}}
	case "k","up","shift+up":if m.tab==1{items:=m.items();if k=="shift+up"&&len(items)>0{m.selected[items[m.cursor].ID]=true};m.cursor--;m.clamp();if k=="shift+up"&&len(items)>0{m.selected[items[m.cursor].ID]=true}}
	case "space"," ":if m.tab==1{items:=m.items();if len(items)>0{id:=items[m.cursor].ID;m.selected[id]=!m.selected[id]}}
	case "i","enter","tab":if m.tab==1{return m.inspect()}
	case "d":if m.tab==1&&len(m.removalIDs())>0{m.overlay="remove"}
	}
	return nil
}

func(m *Model)overlayKey(msg tea.KeyPressMsg)tea.Cmd{
	k:=msg.String();if k=="esc"{m.closeOverlay();return nil}
	switch m.overlay{
	case "help":if k=="?"||k=="q"{m.closeOverlay()};return nil
	case "remove":
		if k=="y"{ids:=m.removalIDs();m.closeOverlay();return func()tea.Msg{return removedMsg(m.service.Remove(m.ctx,ids))}}
		if k=="n"||k=="q"{m.closeOverlay()};return nil
	case "pull":
		if k=="x"{m.jobs.Cancel(m.jobID)};if k=="J"{m.overlay="jobs"};if k=="R"{for _,j:=range m.jobList{if j.ID==m.jobID&&j.State!="running"{return m.startPull(j.Target)}}};return nil
	case "jobs":
		if k=="enter"&&len(m.jobList)>0{m.jobID=m.jobList[len(m.jobList)-1].ID;m.overlay="pull"};return nil
	case "pull-input":if k=="enter"{ref,err:=services.NormalizeReference(m.input.Value());if err!=nil{return m.notify(err.Error())};return m.startPull(ref)}
	case "filter":if k=="enter"{m.filter=m.input.Value();m.cursor=0;m.closeOverlay();return nil}
	case "palette":
		results:=m.paletteActions();if k=="down"||k=="ctrl+n"{m.paletteCursor=min(max(0,len(results)-1),m.paletteCursor+1);return nil};if k=="up"{m.paletteCursor=max(0,m.paletteCursor-1);return nil}
		if k=="enter"&&len(results)>0{index:=min(m.paletteCursor,len(results)-1);action:=results[index];m.closeOverlay();if strings.HasPrefix(action.key,"image:"){id:=strings.TrimPrefix(action.key,"image:");m.tab=1;for n,i:=range m.items(){if i.ID==id{m.cursor=n;return m.inspect()}};return nil};return m.key(tea.KeyPressMsg{Code:[]rune(action.key)[0],Text:action.key})}
	}
	var cmd tea.Cmd;m.input,cmd=m.input.Update(msg);m.paletteCursor=0;return cmd
}

func(m *Model)startPull(ref string)tea.Cmd{
	m.input.Blur();m.overlay="pull"
	return func()tea.Msg{id:=m.jobs.Start(m.ctx,m.service,ref);return pullStartedMsg{id}}
}
type pullStartedMsg struct{id uint64}

func(m *Model)removalIDs()[]string{
	ids:=[]string{};for _,i:=range m.store.Images{if m.selected[i.ID]{ids=append(ids,i.ID)}}
	if len(ids)==0{items:=m.items();if len(items)>0{ids=append(ids,items[m.cursor].ID)}};return ids
}

func(m *Model)mouse(x,y int)tea.Cmd{
	if m.overlay!=""{return nil};if y==1{offset:=0;for n,name:=range components.TabNames(){label:=fmt.Sprintf("%d %s",n+1,name);if m.width<tokens.DetailWidth{label=fmt.Sprintf("%d %.3s",n+1,name)};end:=offset+len(label)+tokens.Gap;if x>=offset&&x<end{m.tab=n;m.detail=false;return nil};offset=end}}
	if m.tab==1&&y>=tokens.HeaderHeight+tokens.PanelFrame+1{row:=y-tokens.HeaderHeight-tokens.PanelFrame-1;capacity:=max(1,m.height-tokens.HeaderHeight-tokens.FooterHeight-tokens.PanelFrame-tokens.InsetY*2);start:=max(0,m.cursor-capacity+1);index:=start+row;if index<len(m.items()){m.cursor=index}}
	return nil
}
