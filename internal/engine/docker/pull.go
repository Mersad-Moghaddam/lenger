package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
	"github.com/OWNER/lenger/internal/engine"
)

// DecodePull parses progress and embedded errors; malformed streams never succeed.
func DecodePull(ctx context.Context,r io.Reader,emit func(engine.Progress)error)error{
	decoder:=json.NewDecoder(r)
	for{
		if err:=ctx.Err();err!=nil{return err}
		var msg struct{
			ID string `json:"id"`
			Status string `json:"status"`
			Error string `json:"error"`
			ErrorDetail struct{Message string `json:"message"`} `json:"errorDetail"`
			Progress struct{Current,Total int64} `json:"progressDetail"`
		}
		if err:=decoder.Decode(&msg);err!=nil{if errors.Is(err,io.EOF){return nil};return fmt.Errorf("decode pull progress: %w",err)}
		if msg.Error!=""{return engine.Classify(errors.New(msg.Error))};if msg.ErrorDetail.Message!=""{return engine.Classify(errors.New(msg.ErrorDetail.Message))}
		if msg.ID==""{continue}
		if err:=emit(engine.Progress{ID:msg.ID,Status:msg.Status,Current:msg.Progress.Current,Total:msg.Progress.Total});err!=nil{return err}
	}
}

func registryAuth(ctx context.Context,ref string)(string,error){
	if err:=ctx.Err();err!=nil{return "",err}
	named,err:=reference.ParseNormalizedNamed(ref);if err!=nil{return "",fmt.Errorf("invalid reference: %w",err)}
	host:=reference.Domain(named);if host=="docker.io"{host="https://index.docker.io/v1/"}
	// Docker's credential helper protocol has no context parameter. Resolve it in
	// the job goroutine, never Update, and do not persist or log the result.
	cfg,err:=config.Load(config.Dir());if err!=nil{return "",fmt.Errorf("load Docker config: %w",err)}
	auth,err:=cfg.GetAuthConfig(host);if err!=nil{return "",fmt.Errorf("credential helper for %s: %w",host,err)}
	if err:=ctx.Err();err!=nil{return "",err}
	encoded,err:=registry.EncodeAuthConfig(registry.AuthConfig{Username:auth.Username,Password:auth.Password,Auth:auth.Auth,ServerAddress:auth.ServerAddress,IdentityToken:auth.IdentityToken,RegistryToken:auth.RegistryToken})
	return strings.TrimSpace(encoded),err
}
