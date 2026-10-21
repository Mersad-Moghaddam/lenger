// Package config loads user configuration without writing or storing secrets.
package config

import(
	"fmt"
	"os"
	"path/filepath"
	"github.com/pelletier/go-toml/v2"
)

// Config contains the user-facing TOML settings.
type Config struct{
	UI struct{Theme string `toml:"theme"`;ASCII bool `toml:"ascii_icons"`;ReduceMotion bool `toml:"reduce_motion"`;Mouse bool `toml:"mouse"`} `toml:"ui"`
	Docker struct{Context string `toml:"context"`;SafetyPoll int `toml:"safety_poll_seconds"`} `toml:"docker"`
	Registry struct{Mirrors []string `toml:"mirrors"`;UpdateCheck bool `toml:"update_check"`;TTL int `toml:"update_check_ttl_hours"`;Scanner string `toml:"scanner"`} `toml:"registry"`
	Logs struct{Buffer int `toml:"buffer_lines"`;Timestamps bool `toml:"show_timestamps"`} `toml:"logs"`
	Keys map[string]string `toml:"keys"`
}

// Defaults returns an independent configuration with no mutable shared state.
func Defaults()Config{var c Config;c.UI.Theme="omarchy";c.UI.Mouse=true;c.Docker.SafetyPoll=30;c.Registry.UpdateCheck=true;c.Registry.TTL=6;c.Registry.Scanner="auto";c.Logs.Buffer=50000;c.Keys=make(map[string]string);return c}

// Load reads the XDG configuration, treating a missing file as defaults.
func Load()(Config,error){
	c:=Defaults();dir,err:=os.UserConfigDir();if err!=nil{return c,fmt.Errorf("find config directory: %w",err)}
	data,err:=os.ReadFile(filepath.Join(dir,"lenger","config.toml"));if err!=nil&&!os.IsNotExist(err){return c,fmt.Errorf("read config: %w",err)}
	if err==nil{if err=toml.Unmarshal(data,&c);err!=nil{return c,fmt.Errorf("parse config: %w",err)}}
	if os.Getenv("NO_MOTION")=="1"||os.Getenv("SSH_CONNECTION")!=""{c.UI.ReduceMotion=true}
	if os.Getenv("TERM")=="dumb"{c.UI.ASCII=true;c.UI.ReduceMotion=true}
	return c,c.Validate()
}

// Validate rejects invalid limits and conflicting reserved keybindings.
func(c Config)Validate()error{
	if c.Docker.SafetyPoll<5||c.Docker.SafetyPoll>3600{return fmt.Errorf("docker.safety_poll_seconds must be between 5 and 3600")}
	if c.Logs.Buffer<1||c.Logs.Buffer>1_000_000{return fmt.Errorf("logs.buffer_lines must be between 1 and 1000000")}
	if c.Registry.TTL<1{return fmt.Errorf("registry.update_check_ttl_hours must be positive")}
	seen:=make(map[string]string);for action,key:=range c.Keys{if key==""||key=="ctrl+c"||key=="esc"{return fmt.Errorf("invalid binding for %s",action)};if previous,ok:=seen[key];ok{return fmt.Errorf("binding %q conflicts: %s and %s",key,previous,action)};seen[key]=action}
	return nil
}
