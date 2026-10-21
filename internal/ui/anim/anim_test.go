package anim

import "testing"

func TestIdleSchedulesNothing(t *testing.T){s:=Spring{};if s.Next()!=nil{t.Fatal("idle spring scheduled a tick")}}
func TestReducedMotionSchedulesNothing(t *testing.T){s:=Spring{Reduced:true};if s.Move(10)!=nil||s.Value!=10{t.Fatal("reduced motion did not settle instantly")}}
func TestSpringSettles(t *testing.T){s:=Spring{};_ = s.Move(10);for n:=0;n<20;n++{s.Step()};if s.Value!=10||s.Next()!=nil{t.Fatal("finite spring did not settle")}}
