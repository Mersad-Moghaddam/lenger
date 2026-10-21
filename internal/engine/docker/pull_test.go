package docker

import("context";"errors";"strings";"testing";"github.com/OWNER/lenger/internal/engine")

func TestDecodePull(t *testing.T){for _,tc:=range []struct{name,input string;wantError bool;wantCount int}{{"progress","{\"id\":\"layer\",\"status\":\"Downloading\",\"progressDetail\":{\"current\":5,\"total\":10}}",false,1},{"malformed","{broken",true,0},{"embedded error","{\"error\":\"unauthorized\"}",true,0},{"daemon status","{\"status\":\"Digest: sha256:abc\"}",false,0}}{t.Run(tc.name,func(t *testing.T){count:=0;err:=DecodePull(context.Background(),strings.NewReader(tc.input),func(p engine.Progress)error{count++;if p.Current!=5||p.Total!=10{t.Fatalf("bad progress: %+v",p)};return nil});if(err!=nil)!=tc.wantError||count!=tc.wantCount{t.Fatalf("err=%v count=%d",err,count)}})}}

func TestDecodePullCancellation(t *testing.T){ctx,cancel:=context.WithCancel(context.Background());cancel();err:=DecodePull(ctx,strings.NewReader(""),func(engine.Progress)error{return nil});if !errors.Is(err,context.Canceled){t.Fatalf("got %v",err)}}
