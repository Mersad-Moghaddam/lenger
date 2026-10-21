package services

import("context";"strings";"testing";"github.com/OWNER/lenger/internal/engine/fake")

func TestNormalizeReference(t *testing.T){for _,tc:=range []struct{input,want string;bad bool}{{"redis","redis:latest",false},{"localhost:5000/team/app:v1","localhost:5000/team/app:v1",false},{"", "",true},{"bad name","",true},{" redis","",true}}{t.Run(tc.input,func(t *testing.T){got,err:=NormalizeReference(tc.input);if(err!=nil)!=tc.bad||got!=tc.want{t.Fatalf("got %q, %v",got,err)}})}}

func TestInspectMasksSecrets(t *testing.T){ctx:=context.Background();s:=New(fake.New());batch,err:=s.Snapshot(ctx);if err!=nil{t.Fatal(err)};i,err:=s.Inspect(ctx,batch.Images[0].ID);if err!=nil{t.Fatal(err)};for _,v:=range i.Env{if strings.Contains(v,"fixture-secret")||!strings.HasSuffix(v,"=••••"){t.Fatalf("unmasked environment: %s",v)}};for _,v:=range i.Labels{if v!="••••"{t.Fatal("unmasked label")}}}

func TestRemoveInUseImageRefused(t *testing.T){ctx:=context.Background();s:=New(fake.New());batch,err:=s.Snapshot(ctx);if err!=nil{t.Fatal(err)};id:=batch.Images[0].ID;if s.Remove(ctx,[]string{id})[id]==nil{t.Fatal("in-use image was removed")};after,err:=s.Snapshot(ctx);if err!=nil{t.Fatal(err)};if len(after.Images)!=len(batch.Images){t.Fatal("image count changed")}}
