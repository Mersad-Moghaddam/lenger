package app

import(
	"fmt"
	"strings"
	tea "charm.land/bubbletea/v2"
	"github.com/OWNER/lenger/internal/engine"
	"github.com/OWNER/lenger/internal/ui/components"
	"github.com/OWNER/lenger/internal/ui/style"
	"github.com/OWNER/lenger/internal/ui/tokens"
	"github.com/OWNER/lenger/internal/ui/views"
)

// View composes one bounded frame and opts into the alternate screen.
func(m *Model)View()tea.View{
	v:=tea.NewView(m.Render());v.AltScreen=true;v.WindowTitle="Lenger · Anchor your containers";if m.config.UI.Mouse{v.MouseMode=tea.MouseModeCellMotion};return v
}

// Render returns the same screen used by the interactive app and offline gallery.
func(m *Model)Render()string{
	w,h:=m.width,m.height;s:=m.sheet
	if w<tokens.MinWidth||h<tokens.MinHeight{return style.Fit(fmt.Sprintf("Lenger needs at least %d×%d cells.\nCurrent size: %d×%d. Resize or press q to quit.",tokens.MinWidth,tokens.MinHeight,w,h),w,h)}
	status:="docker · local";if m.fixture{status="fixture engine · offline"};if m.loading{status="connecting…"};if m.err!=nil{status="docker · unavailable"}
	header:=s.Gradient(tokens.Icons(s.ASCII).Anchor+" LENGER")+"   "+s.Muted("Anchor your containers.")+"   "+s.Muted(status)+"\n"+components.Tabs(s,m.tab,w)+"\n"
	height:=h-tokens.HeaderHeight-tokens.FooterHeight;var body string
	switch{
	case m.loading:body=s.Panel("CONNECTING","Loading local images and containers…\nThe interface remains available while Docker responds.",w,height,false)
	case m.err!=nil:body=s.Panel("DOCKER NEEDS ATTENTION",m.err.Message+"\n\n"+m.err.Suggestion+"\n\nR retry   q quit\nFor an offline preview: LENGER_ENGINE=fake lenger",w,height,true)
	case m.tab==0:body=views.Dashboard(s,m.store,m.eventHistory,w,height,m.fixture)
	case m.tab==1:
		items:=m.items();var inspect *engine.Inspection;if len(items)>0{if i,ok:=m.inspections[items[m.cursor].ID];ok{inspect=&i}}
		body=views.Images(s,items,m.cursor,m.selected,inspect,m.detail,[]string{"name","size","age"}[m.sortMode],m.filter,w,height,m.clock)
	default:requirements:=[]string{"","","CTR-01–CTR-05 (M1)","PRJ-01–PRJ-05 (M2)","VOL-01 (M1)","NET-01 (M2)","SYS-01–SYS-04 (M1/M2)"};body=views.Stub(s,components.TabNames()[m.tab],requirements[m.tab],w,height)
	}
	footer:="? help   : palette   p pull   / filter   J jobs   q quit";if m.tab==1{footer="p pull  i inspect  d remove  o sort  / filter  space select  J jobs"}
	running:=0;for _,j:=range m.jobList{if j.State=="running"{running++}};footer+=fmt.Sprintf("   %s %d jobs",tokens.Icons(s.ASCII).Pull,running)
	second:=m.toast;if second==""{second="Local daemon. No accounts. No telemetry."};base:=style.Fit(header+style.Fit(body,w,height)+"\n"+style.Fit(s.Muted(footer)+"\n"+s.Muted(style.Clean(second)),w,tokens.FooterHeight),w,h)
	if m.overlay!=""{title,content:=m.overlayContent();return components.Modal(s,base,title,content,w,h)};return base
}

func(m *Model)overlayContent()(string,string){
	s:=m.sheet
	switch m.overlay{
	case "help":return "Keyboard chart",components.Help()
	case "pull-input":return "Pull image",m.input.View()+"\n\nUse repository:tag or repository@sha256:digest.\nDocker's configured credential helpers are respected.\n\nenter pull   esc back"
	case "filter":return "Filter images",m.input.View()+"\n\nMatches repository, tag or image ID.\nenter apply   esc back"
	case "palette":
		lines:=[]string{m.input.View(),""};results:=m.paletteActions();start:=max(0,m.paletteCursor-7);for n:=start;n<min(len(results),start+8);n++{prefix:="  ";if n==m.paletteCursor{prefix=tokens.Icons(s.ASCII).Cursor+" "};lines=append(lines,prefix+style.Clean(results[n].name))};if len(results)>0{index:=min(m.paletteCursor,len(results)-1);lines=append(lines,"",s.Muted(style.Clean(results[index].command)))};if len(results)==0{lines=append(lines,"No matching actions or images.")};lines=append(lines,"","↑/↓ choose   enter run   esc close");return "Command palette",strings.Join(lines,"\n")
	case "pull":for _,j:=range m.jobList{if j.ID==m.jobID{return "Pull image",components.Pull(s,j,min(m.width,tokens.ModalWidth)-tokens.InsetX*2-tokens.PanelFrame)}};return "Pull image","Starting pull…\n\nesc hide   J jobs"
	case "jobs":lines:=[]string{};for _,j:=range m.jobList{lines=append(lines,fmt.Sprintf("#%d %s %3.0f%% %s",j.ID,style.Clean(j.Target),j.Fraction()*100,components.Pill(s,j.State)))};if len(lines)==0{lines=append(lines,"No jobs yet. Press p to pull an image.")};return "Session jobs",strings.Join(lines,"\n")+"\n\nenter open most recent   esc close"
	case "remove":
		ids:=m.removalIDs();lines:=[]string{"Remove the following local images?",""};for _,id:=range ids{for _,i:=range m.store.Images{if i.ID==id{lines=append(lines,style.Clean(i.Name())+"  "+components.Bytes(i.Size));for _,c:=range i.UsedBy{lines=append(lines,"  IN USE by "+style.Clean(c.Name)+" — removal will be refused")}}}};lines=append(lines,"","No force flag. Containers and volumes are not removed.","Equivalent: docker image rm "+strings.Join(ids," "),"","y confirm   n / esc cancel");return "Confirm image removal",strings.Join(lines,"\n")
	}
	return "Lenger","esc close"
}
